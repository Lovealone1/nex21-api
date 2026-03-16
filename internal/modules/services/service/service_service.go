package service

import (
	"context"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/services/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type CreateServiceInput struct {
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	SKU             *string `json:"sku,omitempty"`
	DurationMinutes int     `json:"duration_minutes,omitempty"`
	BufferMinutes   int     `json:"buffer_minutes,omitempty"`
	Price           float64 `json:"price"`
	Currency        string  `json:"currency,omitempty"`
	Category        *string `json:"category,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

type UpdateServiceInput struct {
	Name            *string  `json:"name,omitempty"`
	Description     *string  `json:"description,omitempty"`
	SKU             *string  `json:"sku,omitempty"`
	DurationMinutes *int     `json:"duration_minutes,omitempty"`
	BufferMinutes   *int     `json:"buffer_minutes,omitempty"`
	Price           *float64 `json:"price,omitempty"`
	Currency        *string  `json:"currency,omitempty"`
	Category        *string  `json:"category,omitempty"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

type ServiceService interface {
	CreateService(ctx context.Context, tenantID string, input CreateServiceInput) (*repo.Service, error)
	GetServiceByID(ctx context.Context, tenantID, id string) (*repo.Service, error)
	UpdateService(ctx context.Context, tenantID, id string, input UpdateServiceInput) (*repo.Service, error)
	DeleteService(ctx context.Context, tenantID, id string) error
	ListServices(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Service], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Service, error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

type serviceService struct {
	repo repo.ServiceRepo
}

func NewServiceService(r repo.ServiceRepo) ServiceService {
	return &serviceService{repo: r}
}

func (s *serviceService) CreateService(ctx context.Context, tenantID string, input CreateServiceInput) (*repo.Service, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ServiceService.Create", "tenantID is required")
	}
	if input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "ServiceService.Create", "name is required")
	}
	if input.Currency == "" {
		input.Currency = "COP"
	}
	if input.DurationMinutes <= 0 {
		input.DurationMinutes = 30
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	svc := &repo.Service{
		TenantID:        tenantID,
		Name:            input.Name,
		Description:     input.Description,
		SKU:             input.SKU,
		DurationMinutes: input.DurationMinutes,
		BufferMinutes:   input.BufferMinutes,
		Price:           input.Price,
		Currency:        input.Currency,
		Category:        input.Category,
		IsActive:        isActive,
	}

	if err := s.repo.Create(ctx, svc); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ServiceService.Create", "Failed to create service")
	}

	return svc, nil
}

func (s *serviceService) GetServiceByID(ctx context.Context, tenantID, id string) (*repo.Service, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ServiceService.GetByID", "id and tenantID are required")
	}

	svc, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ServiceService.GetByID", "Service not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ServiceService.GetByID", "Failed to fetch service")
	}

	return svc, nil
}

func (s *serviceService) UpdateService(ctx context.Context, tenantID, id string, input UpdateServiceInput) (*repo.Service, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ServiceService.Update", "id and tenantID are required")
	}

	if input.Name != nil && *input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "ServiceService.Update", "name cannot be empty")
	}

	svc, err := s.repo.Update(ctx, tenantID, id, repo.UpdateFields{
		Name:            input.Name,
		Description:     input.Description,
		SKU:             input.SKU,
		DurationMinutes: input.DurationMinutes,
		BufferMinutes:   input.BufferMinutes,
		Price:           input.Price,
		Currency:        input.Currency,
		Category:        input.Category,
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ServiceService.Update", "Service not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ServiceService.Update", "Failed to update service")
	}

	return svc, nil
}

func (s *serviceService) DeleteService(ctx context.Context, tenantID, id string) error {
	if id == "" || tenantID == "" {
		return errors.New(errors.InvalidArgument, "ServiceService.Delete", "id and tenantID are required")
	}

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "ServiceService.Delete", "Service not found")
		}
		return errors.Wrap(err, errors.Internal, "ServiceService.Delete", "Failed to delete service")
	}

	return nil
}

func (s *serviceService) ListServices(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Service], error) {
	if tenantID == "" {
		return store.ResultList[repo.Service]{}, errors.New(errors.InvalidArgument, "ServiceService.List", "tenantID is required")
	}

	result, err := s.repo.List(ctx, tenantID, page)
	if err != nil {
		return store.ResultList[repo.Service]{}, errors.Wrap(err, errors.Internal, "ServiceService.List", "Failed to list services")
	}

	return result, nil
}

func (s *serviceService) ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Service, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ServiceService.ToggleStatus", "id and tenantID are required")
	}

	svc, err := s.repo.ToggleStatus(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ServiceService.ToggleStatus", "Service not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ServiceService.ToggleStatus", "Failed to toggle service status")
	}

	return svc, nil
}
