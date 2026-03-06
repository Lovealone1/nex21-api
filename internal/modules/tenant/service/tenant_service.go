package service

import (
	"context"
	"fmt"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/tenant/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

// ─── Valid values ─────────────────────────────────────────────────────────────

var validPlans = map[string]bool{
	"free":       true,
	"starter":    true,
	"pro":        true,
	"enterprise": true,
}

// ─── DTOs ────────────────────────────────────────────────────────────────────

// CreateTenantInput contains the required and optional fields for creating a tenant.
type CreateTenantInput struct {
	// Name is the human-readable display name (required).
	Name string `json:"name"`
	// Slug is the URL-safe identifier, e.g. "acme-corp" (required, unique).
	Slug string `json:"slug"`
	// Plan is the billing plan. Defaults to "free" if empty.
	// Valid values: free, starter, pro, enterprise.
	Plan string `json:"plan,omitempty"`
	// IsActive is the tenant's lifecycle status. Defaults to true if empty.
	IsActive *bool `json:"is_active,omitempty"`
}

// UpdateTenantInput contains the optional fields for patching a tenant.
// Only non-nil fields are applied — omit any field you don't want to change.
type UpdateTenantInput struct {
	Name     *string `json:"name,omitempty"`
	Slug     *string `json:"slug,omitempty"`
	Plan     *string `json:"plan,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

// TenantService defines the use cases for managing tenants.
type TenantService interface {
	// CreateTenant registers a new tenant.
	CreateTenant(ctx context.Context, input CreateTenantInput) (*repo.Tenant, error)
	// GetTenantByID fetches a single tenant by its UUID.
	GetTenantByID(ctx context.Context, id string) (*repo.Tenant, error)
	// UpdateTenant applies a partial patch to a tenant.
	UpdateTenant(ctx context.Context, id string, input UpdateTenantInput) (*repo.Tenant, error)
	// DeleteTenant permanently removes a tenant.
	DeleteTenant(ctx context.Context, id string) error
	// ListTenants returns a paginated list of all tenants.
	ListTenants(ctx context.Context, page store.Page) (store.ResultList[repo.Tenant], error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

type tenantService struct {
	repo repo.TenantRepo
}

// NewTenantService wires the service to its repository.
func NewTenantService(r repo.TenantRepo) TenantService {
	return &tenantService{repo: r}
}

func (s *tenantService) CreateTenant(ctx context.Context, input CreateTenantInput) (*repo.Tenant, error) {
	if input.Name == "" || input.Slug == "" {
		return nil, errors.New(errors.InvalidArgument, "TenantService.Create", "name and slug are required")
	}

	if input.Plan == "" {
		input.Plan = "free"
	} else if !validPlans[input.Plan] {
		return nil, errors.New(errors.InvalidArgument, "TenantService.Create",
			fmt.Sprintf("invalid plan %q: must be one of free, starter, pro, enterprise", input.Plan))
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	t := &repo.Tenant{
		Name:     input.Name,
		Slug:     input.Slug,
		Plan:     input.Plan,
		IsActive: isActive,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "TenantService.Create", "Failed to create tenant")
	}

	return t, nil
}

func (s *tenantService) GetTenantByID(ctx context.Context, id string) (*repo.Tenant, error) {
	if id == "" {
		return nil, errors.New(errors.InvalidArgument, "TenantService.GetByID", "id is required")
	}

	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "TenantService.GetByID", "Tenant not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "TenantService.GetByID", "Failed to fetch tenant")
	}

	return t, nil
}

func (s *tenantService) UpdateTenant(ctx context.Context, id string, input UpdateTenantInput) (*repo.Tenant, error) {
	if id == "" {
		return nil, errors.New(errors.InvalidArgument, "TenantService.Update", "id is required")
	}
	if input.Name == nil && input.Slug == nil && input.Plan == nil && input.IsActive == nil {
		return nil, errors.New(errors.InvalidArgument, "TenantService.Update", "at least one field must be provided")
	}

	if input.Slug != nil && *input.Slug == "" {
		return nil, errors.New(errors.InvalidArgument, "TenantService.Update", "slug must not be empty")
	}

	if input.Plan != nil && !validPlans[*input.Plan] {
		return nil, errors.New(errors.InvalidArgument, "TenantService.Update",
			fmt.Sprintf("invalid plan %q: must be one of free, starter, pro, enterprise", *input.Plan))
	}

	t, err := s.repo.Update(ctx, id, repo.UpdateFields{
		Name:     input.Name,
		Slug:     input.Slug,
		Plan:     input.Plan,
		IsActive: input.IsActive,
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "TenantService.Update", "Tenant not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "TenantService.Update", "Failed to update tenant")
	}

	return t, nil
}

func (s *tenantService) DeleteTenant(ctx context.Context, id string) error {
	if id == "" {
		return errors.New(errors.InvalidArgument, "TenantService.Delete", "id is required")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "TenantService.Delete", "Tenant not found")
		}
		return errors.Wrap(err, errors.Internal, "TenantService.Delete", "Failed to delete tenant")
	}

	return nil
}

func (s *tenantService) ListTenants(ctx context.Context, page store.Page) (store.ResultList[repo.Tenant], error) {
	result, err := s.repo.List(ctx, page)
	if err != nil {
		return store.ResultList[repo.Tenant]{}, errors.Wrap(err, errors.Internal, "TenantService.List", "Failed to list tenants")
	}

	return result, nil
}
