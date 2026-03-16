package service

import (
	"context"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/locations/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type CreateLocationInput struct {
	Name      string  `json:"name"`
	Code      *string `json:"code,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Email     *string `json:"email,omitempty"`
	Address   *string `json:"address,omitempty"`
	City      *string `json:"city,omitempty"`
	Country   *string `json:"country,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
}

type UpdateLocationInput struct {
	Name      *string `json:"name,omitempty"`
	Code      *string `json:"code,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Email     *string `json:"email,omitempty"`
	Address   *string `json:"address,omitempty"`
	City      *string `json:"city,omitempty"`
	Country   *string `json:"country,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

type LocationService interface {
	CreateLocation(ctx context.Context, tenantID string, input CreateLocationInput) (*repo.Location, error)
	GetLocationByID(ctx context.Context, tenantID, id string) (*repo.Location, error)
	UpdateLocation(ctx context.Context, tenantID, id string, input UpdateLocationInput) (*repo.Location, error)
	DeleteLocation(ctx context.Context, tenantID, id string) error
	ListLocations(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Location], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Location, error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

type locationService struct {
	repo repo.LocationRepo
}

func NewLocationService(r repo.LocationRepo) LocationService {
	return &locationService{repo: r}
}

func (s *locationService) CreateLocation(ctx context.Context, tenantID string, input CreateLocationInput) (*repo.Location, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "LocationService.Create", "tenantID is required")
	}
	if input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "LocationService.Create", "name is required")
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	isDefault := false
	if input.IsDefault != nil {
		isDefault = *input.IsDefault
	}

	l := &repo.Location{
		TenantID:  tenantID,
		Name:      input.Name,
		Code:      input.Code,
		Phone:     input.Phone,
		Email:     input.Email,
		Address:   input.Address,
		City:      input.City,
		Country:   input.Country,
		IsActive:  isActive,
		IsDefault: isDefault,
	}

	if err := s.repo.Create(ctx, l); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "LocationService.Create", "Failed to create location")
	}

	return l, nil
}

func (s *locationService) GetLocationByID(ctx context.Context, tenantID, id string) (*repo.Location, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "LocationService.GetByID", "id and tenantID are required")
	}

	l, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "LocationService.GetByID", "Location not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "LocationService.GetByID", "Failed to fetch location")
	}

	return l, nil
}

func (s *locationService) UpdateLocation(ctx context.Context, tenantID, id string, input UpdateLocationInput) (*repo.Location, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "LocationService.Update", "id and tenantID are required")
	}

	if input.Name != nil && *input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "LocationService.Update", "name cannot be empty")
	}

	l, err := s.repo.Update(ctx, tenantID, id, repo.UpdateFields{
		Name:      input.Name,
		Code:      input.Code,
		Phone:     input.Phone,
		Email:     input.Email,
		Address:   input.Address,
		City:      input.City,
		Country:   input.Country,
		IsActive:  input.IsActive,
		IsDefault: input.IsDefault,
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "LocationService.Update", "Location not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "LocationService.Update", "Failed to update location")
	}

	return l, nil
}

func (s *locationService) DeleteLocation(ctx context.Context, tenantID, id string) error {
	if id == "" || tenantID == "" {
		return errors.New(errors.InvalidArgument, "LocationService.Delete", "id and tenantID are required")
	}

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "LocationService.Delete", "Location not found")
		}
		return errors.Wrap(err, errors.Internal, "LocationService.Delete", "Failed to delete location")
	}

	return nil
}

func (s *locationService) ListLocations(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Location], error) {
	if tenantID == "" {
		return store.ResultList[repo.Location]{}, errors.New(errors.InvalidArgument, "LocationService.List", "tenantID is required")
	}

	result, err := s.repo.List(ctx, tenantID, page)
	if err != nil {
		return store.ResultList[repo.Location]{}, errors.Wrap(err, errors.Internal, "LocationService.List", "Failed to list locations")
	}

	return result, nil
}

func (s *locationService) ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Location, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "LocationService.ToggleStatus", "id and tenantID are required")
	}

	l, err := s.repo.ToggleStatus(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "LocationService.ToggleStatus", "Location not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "LocationService.ToggleStatus", "Failed to toggle location status")
	}

	return l, nil
}
