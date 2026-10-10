package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"

	"ecommerce-microservices/product-service/internal/handler"
	"ecommerce-microservices/product-service/internal/repository"
	"ecommerce-microservices/product-service/internal/service"
	pb "ecommerce-microservices/proto"

	"google.golang.org/grpc"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := os.Getenv("PRODUCT_DB_PATH")
	if dbPath == "" {
		dbPath = "product-service.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open product database: %v", err)
	}
	defer db.Close()

	repo, err := repository.NewSQLiteRepository(context.Background(), db)
	if err != nil {
		log.Fatalf("initialize product database: %v", err)
	}

	listener, err := net.Listen("tcp", ":"+envOrDefault("PRODUCT_GRPC_PORT", "50051"))
	if err != nil {
		log.Fatalf("listen for product gRPC: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterProductServiceServer(server, handler.NewProductServer(service.NewProductService(repo)))
	log.Printf("product service listening on %s", listener.Addr())
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve product gRPC: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
