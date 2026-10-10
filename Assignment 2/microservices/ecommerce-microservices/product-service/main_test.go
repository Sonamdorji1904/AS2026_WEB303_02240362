package main

import (
	"context"
	"database/sql"
	"testing"

	"ecommerce-microservices/product-service/internal/handler"
	"ecommerce-microservices/product-service/internal/repository"
	"ecommerce-microservices/product-service/internal/service"
	pb "ecommerce-microservices/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	_ "modernc.org/sqlite"
)

func TestProductCRUDAndValidation(t *testing.T) {
	db, err := sql.Open("sqlite", "file:product-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo, err := repository.NewSQLiteRepository(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	server := handler.NewProductServer(service.NewProductService(repo))
	ctx := context.Background()

	created, err := server.CreateProduct(ctx, &pb.CreateProductRequest{Name: "Keyboard", Price: 50, Stock: 4})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetId() == "" {
		t.Fatal("created product has no ID")
	}

	fetched, err := server.GetProduct(ctx, &pb.GetProductRequest{Id: created.GetId()})
	if err != nil || fetched.GetName() != "Keyboard" {
		t.Fatalf("get product = %v, err = %v", fetched, err)
	}

	updated, err := server.UpdateProduct(ctx, &pb.UpdateProductRequest{
		Id: created.GetId(), Name: "Updated keyboard", Price: 60, Stock: 3,
	})
	if err != nil || updated.GetPrice() != 60 {
		t.Fatalf("update product = %v, err = %v", updated, err)
	}

	if _, err := server.CreateProduct(ctx, &pb.CreateProductRequest{Name: "", Price: 10, Stock: 1}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid create code = %v", status.Code(err))
	}
	if _, err := server.GetProduct(ctx, &pb.GetProductRequest{Id: "missing"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing get code = %v", status.Code(err))
	}

	if _, err := server.DeleteProduct(ctx, &pb.DeleteProductRequest{Id: created.GetId()}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.GetProduct(ctx, &pb.GetProductRequest{Id: created.GetId()}); status.Code(err) != codes.NotFound {
		t.Fatalf("deleted get code = %v", status.Code(err))
	}
}
