package staffservice

import (
	"context"
	"fmt"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	staffdomain "github.com/Lovealone1/nex21-api/internal/modules/staff/domain"
	staffrepo "github.com/Lovealone1/nex21-api/internal/modules/staff/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type CreateStaffInput struct {
	DisplayName string                `json:"display_name"`
	Email       *string               `json:"email,omitempty"`
	Phone       *string               `json:"phone,omitempty"`
	LocationID  *string               `json:"location_id,omitempty"`
	ProfileID   *string               `json:"profile_id,omitempty"`
	Role        staffdomain.StaffRole `json:"role"`
}

type UpdateStaffInput struct {
	DisplayName *string `json:"display_name,omitempty"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	LocationID  *string `json:"location_id,omitempty"`
	ProfileID   *string `json:"profile_id,omitempty"`
}

type ChangeRoleInput struct {
	Role staffdomain.StaffRole `json:"role"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

type StaffService interface {
	CreateStaff(ctx context.Context, tenantID string, input CreateStaffInput) (*staffrepo.Staff, error)
	GetStaffByID(ctx context.Context, tenantID, id string) (*staffrepo.Staff, error)
	UpdateStaff(ctx context.Context, tenantID, id string, input UpdateStaffInput) (*staffrepo.Staff, error)
	DeleteStaff(ctx context.Context, tenantID, id string) error
	ListStaff(ctx context.Context, tenantID string, page store.Page) (store.ResultList[staffrepo.Staff], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*staffrepo.Staff, error)
	ChangeRole(ctx context.Context, tenantID, id string, input ChangeRoleInput) (*staffrepo.Staff, error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

type staffService struct {
	repo staffrepo.StaffRepo
}

func NewStaffService(r staffrepo.StaffRepo) StaffService {
	return &staffService{repo: r}
}

func (s *staffService) CreateStaff(ctx context.Context, tenantID string, input CreateStaffInput) (*staffrepo.Staff, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffService.Create", "tenantID is required")
	}
	if input.DisplayName == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffService.Create", "display_name is required")
	}
	if !isValidRole(input.Role) {
		return nil, errors.New(errors.InvalidArgument, "StaffService.Create", fmt.Sprintf("invalid role: %s", input.Role))
	}

	st := &staffrepo.Staff{
		TenantID:    tenantID,
		DisplayName: input.DisplayName,
		Email:       input.Email,
		Phone:       input.Phone,
		LocationID:  input.LocationID,
		ProfileID:   input.ProfileID,
		Role:        input.Role,
		IsActive:    true,
	}

	if err := s.repo.Create(ctx, st); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "StaffService.Create", "Failed to create staff member")
	}

	return st, nil
}

func (s *staffService) GetStaffByID(ctx context.Context, tenantID, id string) (*staffrepo.Staff, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffService.GetByID", "id and tenantID are required")
	}

	st, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "StaffService.GetByID", "Staff item not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "StaffService.GetByID", "Failed to fetch staff member")
	}

	return st, nil
}

func (s *staffService) UpdateStaff(ctx context.Context, tenantID, id string, input UpdateStaffInput) (*staffrepo.Staff, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffService.Update", "id and tenantID are required")
	}

	st, err := s.repo.Update(ctx, tenantID, id, staffrepo.UpdateFields{
		DisplayName: input.DisplayName,
		Email:       input.Email,
		Phone:       input.Phone,
		LocationID:  input.LocationID,
		ProfileID:   input.ProfileID,
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "StaffService.Update", "Staff item not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "StaffService.Update", "Failed to update staff member")
	}

	return st, nil
}

func (s *staffService) DeleteStaff(ctx context.Context, tenantID, id string) error {
	if id == "" || tenantID == "" {
		return errors.New(errors.InvalidArgument, "StaffService.Delete", "id and tenantID are required")
	}

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "StaffService.Delete", "Staff item not found")
		}
		return errors.Wrap(err, errors.Internal, "StaffService.Delete", "Failed to delete staff member")
	}

	return nil
}

func (s *staffService) ListStaff(ctx context.Context, tenantID string, page store.Page) (store.ResultList[staffrepo.Staff], error) {
	if tenantID == "" {
		return store.ResultList[staffrepo.Staff]{}, errors.New(errors.InvalidArgument, "StaffService.List", "tenantID is required")
	}

	result, err := s.repo.List(ctx, tenantID, page)
	if err != nil {
		return store.ResultList[staffrepo.Staff]{}, errors.Wrap(err, errors.Internal, "StaffService.List", "Failed to list staff")
	}

	return result, nil
}

func (s *staffService) ToggleStatus(ctx context.Context, tenantID, id string) (*staffrepo.Staff, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffService.ToggleStatus", "id and tenantID are required")
	}

	st, err := s.repo.ToggleStatus(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "StaffService.ToggleStatus", "Staff item not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "StaffService.ToggleStatus", "Failed to toggle staff status")
	}

	return st, nil
}

func (s *staffService) ChangeRole(ctx context.Context, tenantID, id string, input ChangeRoleInput) (*staffrepo.Staff, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffService.ChangeRole", "id and tenantID are required")
	}
	if !isValidRole(input.Role) {
		return nil, errors.New(errors.InvalidArgument, "StaffService.ChangeRole", fmt.Sprintf("invalid role: %s", input.Role))
	}

	st, err := s.repo.Update(ctx, tenantID, id, staffrepo.UpdateFields{
		Role: &input.Role,
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "StaffService.ChangeRole", "Staff item not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "StaffService.ChangeRole", "Failed to change staff role")
	}

	return st, nil
}

// Helpers

func isValidRole(r staffdomain.StaffRole) bool {
	switch r {
	case staffdomain.RoleOwner, staffdomain.RoleAdmin, staffdomain.RoleManager, staffdomain.RoleStaff, staffdomain.RoleReceptionist:
		return true
	}
	return false
}
