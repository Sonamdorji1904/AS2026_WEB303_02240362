package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pb "ecommerce-microservices/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockProductClient struct {
	create func(context.Context, *pb.CreateProductRequest, ...grpc.CallOption) (*pb.Product, error)
	get    func(context.Context, *pb.GetProductRequest, ...grpc.CallOption) (*pb.Product, error)
	update func(context.Context, *pb.UpdateProductRequest, ...grpc.CallOption) (*pb.Product, error)
	delete func(context.Context, *pb.DeleteProductRequest, ...grpc.CallOption) (*pb.DeleteProductResponse, error)
}

func (m mockProductClient) CreateProduct(ctx context.Context, req *pb.CreateProductRequest, opts ...grpc.CallOption) (*pb.Product, error) {
	return m.create(ctx, req, opts...)
}
func (m mockProductClient) GetProduct(ctx context.Context, req *pb.GetProductRequest, opts ...grpc.CallOption) (*pb.Product, error) {
	return m.get(ctx, req, opts...)
}
func (m mockProductClient) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest, opts ...grpc.CallOption) (*pb.Product, error) {
	return m.update(ctx, req, opts...)
}
func (m mockProductClient) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest, opts ...grpc.CallOption) (*pb.DeleteProductResponse, error) {
	return m.delete(ctx, req, opts...)
}

type mockOrderClient struct {
	create func(context.Context, *pb.CreateOrderRequest, ...grpc.CallOption) (*pb.Order, error)
	get    func(context.Context, *pb.GetOrderRequest, ...grpc.CallOption) (*pb.Order, error)
	update func(context.Context, *pb.UpdateOrderRequest, ...grpc.CallOption) (*pb.Order, error)
	delete func(context.Context, *pb.DeleteOrderRequest, ...grpc.CallOption) (*pb.DeleteOrderResponse, error)
}

func (m mockOrderClient) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest, opts ...grpc.CallOption) (*pb.Order, error) {
	return m.create(ctx, req, opts...)
}
func (m mockOrderClient) GetOrder(ctx context.Context, req *pb.GetOrderRequest, opts ...grpc.CallOption) (*pb.Order, error) {
	return m.get(ctx, req, opts...)
}
func (m mockOrderClient) UpdateOrder(ctx context.Context, req *pb.UpdateOrderRequest, opts ...grpc.CallOption) (*pb.Order, error) {
	return m.update(ctx, req, opts...)
}
func (m mockOrderClient) DeleteOrder(ctx context.Context, req *pb.DeleteOrderRequest, opts ...grpc.CallOption) (*pb.DeleteOrderResponse, error) {
	return m.delete(ctx, req, opts...)
}

func TestGatewayProductCreateAndGet(t *testing.T) {
	products := mockProductClient{
		create: func(_ context.Context, req *pb.CreateProductRequest, _ ...grpc.CallOption) (*pb.Product, error) {
			return &pb.Product{Id: "p1", Name: req.GetName(), Price: req.GetPrice(), Stock: req.GetStock()}, nil
		},
		get: func(_ context.Context, req *pb.GetProductRequest, _ ...grpc.CallOption) (*pb.Product, error) {
			return &pb.Product{Id: req.GetId(), Name: "Keyboard", Price: 50, Stock: 2}, nil
		},
	}
	app := newGateway(products, mockOrderClient{}, 0)

	create := httptest.NewRecorder()
	app.routes().ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(`{"name":"Keyboard","price":50,"stock":2}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", create.Code)
	}

	get := httptest.NewRecorder()
	app.routes().ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/products/p1", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", get.Code)
	}
}

func TestGatewayValidationAndErrorMapping(t *testing.T) {
	orders := mockOrderClient{
		create: func(context.Context, *pb.CreateOrderRequest, ...grpc.CallOption) (*pb.Order, error) {
			return nil, status.Error(codes.Unavailable, "order service unavailable")
		},
	}
	app := newGateway(mockProductClient{}, orders, 0)

	invalid := httptest.NewRecorder()
	app.routes().ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"productId":"","quantity":0}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("validation status = %d, want 400", invalid.Code)
	}

	unavailable := httptest.NewRecorder()
	unavailableRequest := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"productId":"p1","quantity":1}`))
	app.routes().ServeHTTP(unavailable, unavailableRequest)
	if unavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status = %d, want 503", unavailable.Code)
	}
	var body errorResponse
	if err := json.NewDecoder(unavailable.Body).Decode(&body); err != nil || body.Error == "" {
		t.Fatalf("error response = %q, err = %v", unavailable.Body.String(), err)
	}
}

func TestGRPCHTTPStatusMapping(t *testing.T) {
	tests := []struct {
		code codes.Code
		want int
	}{
		{codes.NotFound, http.StatusNotFound},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.Internal, http.StatusInternalServerError},
	}
	for _, test := range tests {
		if got := grpcHTTPStatus(status.Error(test.code, "test")); got != test.want {
			t.Errorf("%s maps to %d, want %d", test.code, got, test.want)
		}
	}
}
