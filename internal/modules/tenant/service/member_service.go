package service

import (
	"context"
	"fmt"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/tenant/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

// MemberRole represents a valid role within a tenant.
// Must stay in sync with the `profile_role` ENUM in Postgres (migration 000002).
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleStaff  MemberRole = "staff"
	MemberRoleMember MemberRole = "member"
)

var validMemberRoles = map[MemberRole]bool{
	MemberRoleOwner:  true,
	MemberRoleAdmin:  true,
	MemberRoleStaff:  true,
	MemberRoleMember: true,
}

// IsValidMemberRole ensures the role string is recognised.
func IsValidMemberRole(r string) bool {
	return validMemberRoles[MemberRole(r)]
}

// ─── DTOs ────────────────────────────────────────────────────────────────────

// AddMemberInput contains the payload for inviting/adding a user to a tenant.
type AddMemberInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // Defaults to "member" if empty
}

// UpdateMemberRoleInput contains the payload for changing a member's role.
type UpdateMemberRoleInput struct {
	Role string `json:"role"` // Required
}

// ─── Interface ───────────────────────────────────────────────────────────────

// MemberService defines the use cases for managing memberships in a tenant.
type MemberService interface {
	AddMember(ctx context.Context, tenantID string, input AddMemberInput) (*repo.Member, error)
	GetMember(ctx context.Context, tenantID, userID string) (*repo.Member, error)
	UpdateRole(ctx context.Context, tenantID, userID string, input UpdateMemberRoleInput) (*repo.Member, error)
	ToggleStatus(ctx context.Context, tenantID, userID string) (*repo.Member, error)
	RemoveMember(ctx context.Context, tenantID, userID string) error
	ListMembers(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Member], error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

type memberService struct {
	repo repo.MemberRepo
}

// NewMemberService wires the service to the member repository.
func NewMemberService(r repo.MemberRepo) MemberService {
	return &memberService{repo: r}
}

func (s *memberService) AddMember(ctx context.Context, tenantID string, input AddMemberInput) (*repo.Member, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "MemberService.Add", "tenant_id is required")
	}
	if input.UserID == "" {
		return nil, errors.New(errors.InvalidArgument, "MemberService.Add", "user_id is required")
	}

	role := input.Role
	if role == "" {
		role = string(MemberRoleMember) // default role
	} else if !IsValidMemberRole(role) {
		return nil, errors.New(errors.InvalidArgument, "MemberService.Add",
			fmt.Sprintf("invalid role %q: must be one of owner, admin, staff, member", role))
	}

	member, err := s.repo.AddMember(ctx, tenantID, input.UserID, role)
	if err != nil {
		// PostgreSQL unique constraint violation on (tenant_id, user_id)
		// gorm/pgx err code extraction omitted for simplicity, but log could show duplicates.
		return nil, errors.Wrap(err, errors.Internal, "MemberService.Add", "Failed to add member to tenant")
	}

	return member, nil
}

func (s *memberService) GetMember(ctx context.Context, tenantID, userID string) (*repo.Member, error) {
	if tenantID == "" || userID == "" {
		return nil, errors.New(errors.InvalidArgument, "MemberService.Get", "tenant_id and user_id are required")
	}

	m, err := s.repo.GetMember(ctx, tenantID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "MemberService.Get", "Member not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "MemberService.Get", "Failed to get member details")
	}

	return m, nil
}

func (s *memberService) UpdateRole(ctx context.Context, tenantID, userID string, input UpdateMemberRoleInput) (*repo.Member, error) {
	if tenantID == "" || userID == "" {
		return nil, errors.New(errors.InvalidArgument, "MemberService.UpdateRole", "tenant_id and user_id are required")
	}
	if !IsValidMemberRole(input.Role) {
		return nil, errors.New(errors.InvalidArgument, "MemberService.UpdateRole",
			fmt.Sprintf("invalid role %q: must be one of owner, admin, staff, member", input.Role))
	}

	member, err := s.repo.UpdateRole(ctx, tenantID, userID, input.Role)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "MemberService.UpdateRole", "Member not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "MemberService.UpdateRole", "Failed to update member role")
	}
	return member, nil
}

func (s *memberService) ToggleStatus(ctx context.Context, tenantID, userID string) (*repo.Member, error) {
	if tenantID == "" || userID == "" {
		return nil, errors.New(errors.InvalidArgument, "MemberService.ToggleStatus", "tenant_id and user_id are required")
	}

	member, err := s.repo.ToggleStatus(ctx, tenantID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "MemberService.ToggleStatus", "Member not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "MemberService.ToggleStatus", "Failed to toggle member status")
	}
	return member, nil
}

func (s *memberService) RemoveMember(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" || userID == "" {
		return errors.New(errors.InvalidArgument, "MemberService.Remove", "tenant_id and user_id are required")
	}

	err := s.repo.RemoveMember(ctx, tenantID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "MemberService.Remove", "Member not found")
		}
		return errors.Wrap(err, errors.Internal, "MemberService.Remove", "Failed to remove member")
	}
	return nil
}

func (s *memberService) ListMembers(ctx context.Context, tenantID string, page store.Page) (store.ResultList[repo.Member], error) {
	if tenantID == "" {
		return store.ResultList[repo.Member]{}, errors.New(errors.InvalidArgument, "MemberService.List", "tenant_id is required")
	}

	result, err := s.repo.ListMembers(ctx, tenantID, page)
	if err != nil {
		return store.ResultList[repo.Member]{}, errors.Wrap(err, errors.Internal, "MemberService.List", "Failed to list members")
	}

	return result, nil
}
