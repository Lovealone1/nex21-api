package service

import (
	"context"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/products/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type CreateProductInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	SKU         *string `json:"sku,omitempty"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency,omitempty"`
	Quantity    int     `json:"quantity,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type UpdateProductInput struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	SKU         *string  `json:"sku,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	Currency    *string  `json:"currency,omitempty"`
}

type UpdateStockInput struct {
	Quantity int `json:"quantity"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

type ProductService interface {
	CreateProduct(ctx context.Context, tenantID string, input CreateProductInput) (*repo.Product, error)
	GetProductByID(ctx context.Context, tenantID, id string) (*repo.Product, error)
	UpdateProduct(ctx context.Context, tenantID, id string, input UpdateProductInput) (*repo.Product, error)
	DeleteProduct(ctx context.Context, tenantID, id string) error
	ListProducts(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Product], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Product, error)

	// Stock management
	SetStock(ctx context.Context, tenantID, id string, quantity int) (*repo.Product, error)
	IncrementStock(ctx context.Context, tenantID, id string, amount int) error
	DecrementStock(ctx context.Context, tenantID, id string, amount int) error
}

// ─── Implementation ──────────────────────────────────────────────────────────

type productService struct {
	repo repo.ProductRepo
}

func NewProductService(r repo.ProductRepo) ProductService {
	return &productService{repo: r}
}

func (s *productService) CreateProduct(ctx context.Context, tenantID string, input CreateProductInput) (*repo.Product, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ProductService.Create", "tenantID is required")
	}
	if input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "ProductService.Create", "name is required")
	}
	if input.Currency == "" {
		input.Currency = "COP"
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	p := &repo.Product{
		TenantID:    tenantID,
		Name:        input.Name,
		Description: input.Description,
		SKU:         input.SKU,
		Price:       input.Price,
		Currency:    input.Currency,
		Quantity:    input.Quantity,
		IsActive:    isActive,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ProductService.Create", "Failed to create product")
	}

	return p, nil
}

func (s *productService) GetProductByID(ctx context.Context, tenantID, id string) (*repo.Product, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ProductService.GetByID", "id and tenantID are required")
	}

	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ProductService.GetByID", "Product not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ProductService.GetByID", "Failed to fetch product")
	}

	return p, nil
}

func (s *productService) UpdateProduct(ctx context.Context, tenantID, id string, input UpdateProductInput) (*repo.Product, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ProductService.Update", "id and tenantID are required")
	}

	if input.Name != nil && *input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "ProductService.Update", "name cannot be empty")
	}

	p, err := s.repo.Update(ctx, tenantID, id, repo.UpdateFields{
		Name:        input.Name,
		Description: input.Description,
		SKU:         input.SKU,
		Price:       input.Price,
		Currency:    input.Currency,
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ProductService.Update", "Product not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ProductService.Update", "Failed to update product")
	}

	return p, nil
}

func (s *productService) DeleteProduct(ctx context.Context, tenantID, id string) error {
	if id == "" || tenantID == "" {
		return errors.New(errors.InvalidArgument, "ProductService.Delete", "id and tenantID are required")
	}

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "ProductService.Delete", "Product not found")
		}
		return errors.Wrap(err, errors.Internal, "ProductService.Delete", "Failed to delete product")
	}

	return nil
}

func (s *productService) ListProducts(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Product], error) {
	if tenantID == "" {
		return store.ResultList[repo.Product]{}, errors.New(errors.InvalidArgument, "ProductService.List", "tenantID is required")
	}

	result, err := s.repo.List(ctx, tenantID, page)
	if err != nil {
		return store.ResultList[repo.Product]{}, errors.Wrap(err, errors.Internal, "ProductService.List", "Failed to list products")
	}

	return result, nil
}

func (s *productService) ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Product, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ProductService.ToggleStatus", "id and tenantID are required")
	}

	p, err := s.repo.ToggleStatus(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ProductService.ToggleStatus", "Product not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ProductService.ToggleStatus", "Failed to toggle product status")
	}

	return p, nil
}

// ─── Stock Management ──────────────────────────────────────────────────────────

func (s *productService) SetStock(ctx context.Context, tenantID, id string, quantity int) (*repo.Product, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ProductService.SetStock", "id and tenantID are required")
	}
	if quantity < 0 {
		return nil, errors.New(errors.InvalidArgument, "ProductService.SetStock", "quantity cannot be negative")
	}

	if err := s.repo.SetStock(ctx, tenantID, id, quantity); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ProductService.SetStock", "Product not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ProductService.SetStock", "Failed to set stock")
	}

	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *productService) IncrementStock(ctx context.Context, tenantID, id string, amount int) error {
	if amount <= 0 {
		return errors.New(errors.InvalidArgument, "ProductService.IncrementStock", "amount must be positive")
	}
	return s.repo.AdjustStock(ctx, tenantID, id, amount)
}

func (s *productService) DecrementStock(ctx context.Context, tenantID, id string, amount int) error {
	if amount <= 0 {
		return errors.New(errors.InvalidArgument, "ProductService.DecrementStock", "amount must be positive")
	}

	// We might want to check current stock here if we don't allow negative stock
	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if p.Quantity < amount {
		return errors.New(errors.FailedPrecondition, "ProductService.DecrementStock", "insufficient stock")
	}

	return s.repo.AdjustStock(ctx, tenantID, id, -amount)
}
