package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"time"

	"ecommerce-microservices/order-service/internal/handler"
	"ecommerce-microservices/order-service/internal/repository"
	"ecommerce-microservices/order-service/internal/service"
	pb "ecommerce-microservices/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := envOrDefault("ORDER_DB_PATH", "order-service.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open order database: %v", err)
	}
	defer db.Close()

	repo, err := repository.NewSQLiteRepository(context.Background(), db)
	if err != nil {
		log.Fatalf("initialize order database: %v", err)
	}

	productAddress := envOrDefault("PRODUCT_SERVICE_ADDRESS", "localhost:50051")
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	productConn, err := grpc.DialContext(dialCtx, productAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	cancel()
	if err != nil {
		log.Fatalf("connect to product service: %v", err)
	}
	defer productConn.Close()

	orderService := service.NewOrderServiceWithConfig(
		repo,
		pb.NewProductServiceClient(productConn),
		service.ResilienceConfigFromEnv(os.Getenv),
	)
	listener, err := net.Listen("tcp", ":"+envOrDefault("ORDER_GRPC_PORT", "50052"))
	if err != nil {
		log.Fatalf("listen for order gRPC: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterOrderServiceServer(server, handler.NewOrderServer(orderService))
	log.Printf("order service listening on %s", listener.Addr())
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve order gRPC: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
