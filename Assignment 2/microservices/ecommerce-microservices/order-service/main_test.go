package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"ecommerce-microservices/order-service/internal/handler"
	"ecommerce-microservices/order-service/internal/repository"
	"ecommerce-microservices/order-service/internal/service"
	pb "ecommerce-microservices/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	_ "modernc.org/sqlite"
)

type fakeProductClient struct {
	product *pb.Product
	err     error
}

func (f fakeProductClient) GetProduct(context.Context, *pb.GetProductRequest, ...grpc.CallOption) (*pb.Product, error) {
	return f.product, f.err
}

type sequenceProductClient struct {
	errs  []error
	calls int
}

func (f *sequenceProductClient) GetProduct(context.Context, *pb.GetProductRequest, ...grpc.CallOption) (*pb.Product, error) {
	index := f.calls
	f.calls++
	if index < len(f.errs) && f.errs[index] != nil {
		return nil, f.errs[index]
	}
	return &pb.Product{Id: "product-1", Price: 50, Stock: 4}, nil
}

func TestOrderCRUDAndProductDependency(t *testing.T) {
	db, err := sql.Open("sqlite", "file:order-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo, err := repository.NewSQLiteRepository(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	server := handler.NewOrderServer(service.NewOrderService(repo, fakeProductClient{
		product: &pb.Product{Id: "product-1", Name: "Keyboard", Price: 50, Stock: 4},
	}))

	order, err := server.CreateOrder(context.Background(), &pb.CreateOrderRequest{ProductId: "product-1", Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	if order.GetTotalPrice() != 100 {
		t.Fatalf("total price = %v, want 100", order.GetTotalPrice())
	}
	if _, err := server.CreateOrder(context.Background(), &pb.CreateOrderRequest{ProductId: "product-1", Quantity: 0}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid create code = %v", status.Code(err))
	}
	if _, err := server.GetOrder(context.Background(), &pb.GetOrderRequest{Id: "missing"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing get code = %v", status.Code(err))
	}
	if _, err := server.DeleteOrder(context.Background(), &pb.DeleteOrderRequest{Id: order.GetId()}); err != nil {
		t.Fatal(err)
	}
}

func TestProductLookupRetriesTransientErrors(t *testing.T) {
	client := &sequenceProductClient{errs: []error{
		status.Error(codes.Unavailable, "product stopped"),
		status.Error(codes.DeadlineExceeded, "product timed out"),
		nil,
	}}
	config := service.DefaultResilienceConfig()
	config.BaseBackoff = time.Millisecond
	config.CallTimeout = time.Second
	config.MaxRetries = 2
	config.BreakerTrips = 5

	db, err := sql.Open("sqlite", "file:retry-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo, err := repository.NewSQLiteRepository(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	server := handler.NewOrderServer(service.NewOrderServiceWithConfig(repo, client, config))

	order, err := server.CreateOrder(context.Background(), &pb.CreateOrderRequest{ProductId: "product-1", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if order.GetId() == "" || client.calls != 3 {
		t.Fatalf("calls = %d, order = %v; want 3 calls and an order", client.calls, order)
	}
}

func TestProductLookupDoesNotRetryNotFound(t *testing.T) {
	client := &sequenceProductClient{errs: []error{
		status.Error(codes.NotFound, "product missing"),
	}}
	config := service.DefaultResilienceConfig()
	config.BaseBackoff = time.Millisecond

	db, err := sql.Open("sqlite", "file:not-found-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo, err := repository.NewSQLiteRepository(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	server := handler.NewOrderServer(service.NewOrderServiceWithConfig(repo, client, config))

	_, err = server.CreateOrder(context.Background(), &pb.CreateOrderRequest{ProductId: "missing", Quantity: 1})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %s, want NotFound", status.Code(err))
	}
	if client.calls != 1 {
		t.Fatalf("calls = %d, want 1", client.calls)
	}
}

func TestProductCircuitBreakerReturnsUnavailableWhenOpen(t *testing.T) {
	client := &sequenceProductClient{errs: []error{
		status.Error(codes.Unavailable, "product stopped"),
	}}
	config := service.DefaultResilienceConfig()
	config.MaxRetries = 0
	config.BreakerTrips = 1
	config.BreakerTimeout = time.Minute

	db, err := sql.Open("sqlite", "file:breaker-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo, err := repository.NewSQLiteRepository(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	server := handler.NewOrderServer(service.NewOrderServiceWithConfig(repo, client, config))
	request := &pb.CreateOrderRequest{ProductId: "product-1", Quantity: 1}

	_, err = server.CreateOrder(context.Background(), request)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("first code = %s, want Unavailable", status.Code(err))
	}
	_, err = server.CreateOrder(context.Background(), request)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("open circuit code = %s, want Unavailable", status.Code(err))
	}
	if client.calls != 1 {
		t.Fatalf("calls = %d, want 1 after circuit opens", client.calls)
	}
}
