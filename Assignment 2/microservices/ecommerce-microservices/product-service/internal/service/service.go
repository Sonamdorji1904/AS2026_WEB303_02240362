package service

import (
	"context"
	"errors"
	"strings"

	"ecommerce-microservices/product-service/internal/repository"
	pb "ecommerce-microservices/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductService struct {
	repository repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repository: repo}
}

func (s *ProductService) Create(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	if err := validateProductFields(req.GetName(), req.GetPrice(), req.GetStock()); err != nil {
		return nil, err
	}
	product := &pb.Product{
		Id:          uuid.NewString(),
		Name:        strings.TrimSpace(req.GetName()),
		Description: strings.TrimSpace(req.GetDescription()),
		Price:       req.GetPrice(),
		Stock:       req.GetStock(),
	}
	if err := s.repository.Create(ctx, product); err != nil {
		return nil, mapRepositoryError(err)
	}
	return product, nil
}

func (s *ProductService) Get(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	id, err := validateID(req.GetId())
	if err != nil {
		return nil, err
	}
	product, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	return product, nil
}

func (s *ProductService) Update(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error) {
	id, err := validateID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := validateProductFields(req.GetName(), req.GetPrice(), req.GetStock()); err != nil {
		return nil, err
	}
	product := &pb.Product{
		Id:          id,
		Name:        strings.TrimSpace(req.GetName()),
		Description: strings.TrimSpace(req.GetDescription()),
		Price:       req.GetPrice(),
		Stock:       req.GetStock(),
	}
	if err := s.repository.Update(ctx, product); err != nil {
		return nil, mapRepositoryError(err)
	}
	return product, nil
}

func (s *ProductService) Delete(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	id, err := validateID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return nil, mapRepositoryError(err)
	}
	return &pb.DeleteProductResponse{Id: id}, nil
}

func validateID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", status.Error(codes.InvalidArgument, "product ID is required")
	}
	return id, nil
}

func validateProductFields(name string, price float64, stock int32) error {
	if strings.TrimSpace(name) == "" {
		return status.Error(codes.InvalidArgument, "product name is required")
	}
	if price <= 0 {
		return status.Error(codes.InvalidArgument, "product price must be positive")
	}
	if stock < 0 {
		return status.Error(codes.InvalidArgument, "product stock cannot be negative")
	}
	return nil
}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return status.Error(codes.NotFound, "product not found")
	case errors.Is(err, repository.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, "product already exists")
	default:
		return status.Error(codes.Internal, "product database error")
	}
}
