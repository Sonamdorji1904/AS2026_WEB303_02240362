package handler

import (
	"context"

	"ecommerce-microservices/order-service/internal/service"
	pb "ecommerce-microservices/proto"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	service *service.OrderService
}

func NewOrderServer(orderService *service.OrderService) *OrderServer {
	return &OrderServer{service: orderService}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	return s.service.Create(ctx, req)
}

func (s *OrderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	return s.service.Get(ctx, req)
}

func (s *OrderServer) UpdateOrder(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.Order, error) {
	return s.service.Update(ctx, req)
}

func (s *OrderServer) DeleteOrder(ctx context.Context, req *pb.DeleteOrderRequest) (*pb.DeleteOrderResponse, error) {
	return s.service.Delete(ctx, req)
}
