package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	pb "ecommerce-microservices/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type gateway struct {
	products pb.ProductServiceClient
	orders   pb.OrderServiceClient
	timeout  time.Duration
}

type productInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int32   `json:"stock"`
}

type orderInput struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	productConn := dial(envOrDefault("PRODUCT_SERVICE_ADDRESS", "localhost:50051"))
	defer productConn.Close()
	orderConn := dial(envOrDefault("ORDER_SERVICE_ADDRESS", "localhost:50052"))
	defer orderConn.Close()

	app := newGateway(
		pb.NewProductServiceClient(productConn),
		pb.NewOrderServiceClient(orderConn),
		durationEnv(os.Getenv("GATEWAY_REQUEST_TIMEOUT"), 5*time.Second),
	)
	log.Printf("API gateway listening on :%s", envOrDefault("GATEWAY_HTTP_PORT", "8080"))
	if err := http.ListenAndServe(":"+envOrDefault("GATEWAY_HTTP_PORT", "8080"), app.routes()); err != nil {
		log.Fatalf("serve API gateway: %v", err)
	}
}

func newGateway(products pb.ProductServiceClient, orders pb.OrderServiceClient, timeout time.Duration) *gateway {
	return &gateway{products: products, orders: orders, timeout: timeout}
}

func (g *gateway) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/products", g.productsCollection)
	mux.HandleFunc("/products/", g.productResource)
	mux.HandleFunc("/orders", g.ordersCollection)
	mux.HandleFunc("/orders/", g.orderResource)
	return mux
}

func (g *gateway) productsCollection(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/products" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var input productInput
		if !decodeJSON(w, r, &input) || !validateProductInput(input) {
			writeError(w, http.StatusBadRequest, "name is required, price must be positive, and stock cannot be negative")
			return
		}
		ctx, cancel := g.requestContext(r)
		defer cancel()
		product, err := g.products.CreateProduct(ctx, &pb.CreateProductRequest{
			Name: input.Name, Description: input.Description, Price: input.Price, Stock: input.Stock,
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, product)
	case http.MethodGet:
		writeError(w, http.StatusBadRequest, "product ID is required; use GET /products/{id}")
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *gateway) productResource(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path, "/products/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "product ID is required")
		return
	}
	ctx, cancel := g.requestContext(r)
	defer cancel()
	switch r.Method {
	case http.MethodGet:
		product, err := g.products.GetProduct(ctx, &pb.GetProductRequest{Id: id})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, product)
	case http.MethodPut:
		var input productInput
		if !decodeJSON(w, r, &input) || !validateProductInput(input) {
			writeError(w, http.StatusBadRequest, "name is required, price must be positive, and stock cannot be negative")
			return
		}
		product, err := g.products.UpdateProduct(ctx, &pb.UpdateProductRequest{
			Id: id, Name: input.Name, Description: input.Description, Price: input.Price, Stock: input.Stock,
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, product)
	case http.MethodDelete:
		if _, err := g.products.DeleteProduct(ctx, &pb.DeleteProductRequest{Id: id}); err != nil {
			writeGRPCError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut+", "+http.MethodDelete)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *gateway) ordersCollection(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/orders" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input orderInput
	if !decodeJSON(w, r, &input) || !validateOrderInput(input) {
		writeError(w, http.StatusBadRequest, "productId is required and quantity must be positive")
		return
	}
	ctx, cancel := g.requestContext(r)
	defer cancel()
	order, err := g.orders.CreateOrder(ctx, &pb.CreateOrderRequest{ProductId: input.ProductID, Quantity: input.Quantity})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (g *gateway) orderResource(w http.ResponseWriter, r *http.Request) {
	id := resourceID(r.URL.Path, "/orders/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "order ID is required")
		return
	}
	ctx, cancel := g.requestContext(r)
	defer cancel()
	switch r.Method {
	case http.MethodGet:
		order, err := g.orders.GetOrder(ctx, &pb.GetOrderRequest{Id: id})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, order)
	case http.MethodPut:
		var input orderInput
		if !decodeJSON(w, r, &input) || !validateOrderInput(input) {
			writeError(w, http.StatusBadRequest, "productId is required and quantity must be positive")
			return
		}
		order, err := g.orders.UpdateOrder(ctx, &pb.UpdateOrderRequest{
			Id: id, ProductId: input.ProductID, Quantity: input.Quantity,
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, order)
	case http.MethodDelete:
		if _, err := g.orders.DeleteOrder(ctx, &pb.DeleteOrderRequest{Id: id}); err != nil {
			writeGRPCError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut+", "+http.MethodDelete)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *gateway) requestContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), g.timeout)
}

func dial(address string) *grpc.ClientConn {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial %s: %v", address, err)
	}
	return conn
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false
	}
	return true
}

func validateProductInput(input productInput) bool {
	return strings.TrimSpace(input.Name) != "" && input.Price > 0 && input.Stock >= 0
}

func validateOrderInput(input orderInput) bool {
	return strings.TrimSpace(input.ProductID) != "" && input.Quantity > 0
}

func resourceID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	if id == path || strings.Contains(id, "/") {
		return ""
	}
	return strings.TrimSpace(id)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	writeError(w, grpcHTTPStatus(err), status.Convert(err).Message())
}

func grpcHTTPStatus(err error) int {
	switch status.Code(err) {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, code int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
