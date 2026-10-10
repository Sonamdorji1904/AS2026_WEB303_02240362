package handler

import (
	"context"

	"ecommerce-microservices/product-service/internal/service"
	pb "ecommerce-microservices/proto"
)

type ProductServer struct {
	pb.UnimplementedProductServiceServer
	service *service.ProductService
}

func NewProductServer(productService *service.ProductService) *ProductServer {
	return &ProductServer{service: productService}
}

func (s *ProductServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	return s.service.Create(ctx, req)
}

func (s *ProductServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	return s.service.Get(ctx, req)
}

func (s *ProductServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error) {
	return s.service.Update(ctx, req)
}

func (s *ProductServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	return s.service.Delete(ctx, req)
}
