package service

import (
	"context"
	"errors"
	"strings"

	"ecommerce-microservices/order-service/internal/repository"
	pb "ecommerce-microservices/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductClient interface {
	GetProduct(context.Context, *pb.GetProductRequest, ...grpc.CallOption) (*pb.Product, error)
}

type OrderService struct {
	repository    repository.OrderRepository
	productClient ProductClient
}

func NewOrderService(repo repository.OrderRepository, productClient ProductClient) *OrderService {
	return &OrderService{
		repository:    repo,
		productClient: NewResilientProductClient(productClient, DefaultResilienceConfig()),
	}
}

func NewOrderServiceWithConfig(repo repository.OrderRepository, productClient ProductClient, config ResilienceConfig) *OrderService {
	return &OrderService{
		repository:    repo,
		productClient: NewResilientProductClient(productClient, config),
	}
}

func (s *OrderService) Create(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	productID, quantity, err := validateOrder(req.GetProductId(), req.GetQuantity())
	if err != nil {
		return nil, err
	}
	product, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: productID})
	if err != nil {
		return nil, err
	}
	if product.GetStock() < quantity {
		return nil, status.Error(codes.FailedPrecondition, "product does not have enough stock")
	}
	order := &pb.Order{
		Id:         uuid.NewString(),
		ProductId:  productID,
		Quantity:   quantity,
		UnitPrice:  product.GetPrice(),
		TotalPrice: product.GetPrice() * float64(quantity),
	}
	if err := s.repository.Create(ctx, order); err != nil {
		return nil, status.Error(codes.Internal, "order database error")
	}
	return order, nil
}

func (s *OrderService) Get(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	id, err := validateID(req.GetId(), "order ID")
	if err != nil {
		return nil, err
	}
	order, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	if _, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: order.GetProductId()}); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *OrderService) Update(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.Order, error) {
	id, err := validateID(req.GetId(), "order ID")
	if err != nil {
		return nil, err
	}
	productID, quantity, err := validateOrder(req.GetProductId(), req.GetQuantity())
	if err != nil {
		return nil, err
	}
	product, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: productID})
	if err != nil {
		return nil, err
	}
	if product.GetStock() < quantity {
		return nil, status.Error(codes.FailedPrecondition, "product does not have enough stock")
	}
	order := &pb.Order{Id: id, ProductId: productID, Quantity: quantity, UnitPrice: product.GetPrice(), TotalPrice: product.GetPrice() * float64(quantity)}
	if err := s.repository.Update(ctx, order); err != nil {
		return nil, mapRepositoryError(err)
	}
	return order, nil
}

func (s *OrderService) Delete(ctx context.Context, req *pb.DeleteOrderRequest) (*pb.DeleteOrderResponse, error) {
	id, err := validateID(req.GetId(), "order ID")
	if err != nil {
		return nil, err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return nil, mapRepositoryError(err)
	}
	return &pb.DeleteOrderResponse{Id: id}, nil
}

func validateOrder(productID string, quantity int32) (string, int32, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return "", 0, status.Error(codes.InvalidArgument, "product ID is required")
	}
	if quantity <= 0 {
		return "", 0, status.Error(codes.InvalidArgument, "order quantity must be positive")
	}
	return productID, quantity, nil
}

func validateID(id, field string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", status.Errorf(codes.InvalidArgument, "%s is required", field)
	}
	return id, nil
}

func mapRepositoryError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return status.Error(codes.NotFound, "order not found")
	}
	return status.Error(codes.Internal, "order database error")
}
