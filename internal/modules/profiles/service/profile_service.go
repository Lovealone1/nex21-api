package service

import (
	"context"
	"fmt"

	"github.com/Lovealone1/nex21-api/internal/modules/auth/application"
	"github.com/Lovealone1/nex21-api/internal/modules/profiles/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
)

// ProfileRole represents a valid role in the system.
// Must stay in sync with the `profile_role` ENUM in Postgres (migration 000002).
type ProfileRole string

const (
	RoleOwner  ProfileRole = "owner"  // Tenant owner – full control
	RoleAdmin  ProfileRole = "admin"  // Tenant admin – manage users & settings
	RoleStaff  ProfileRole = "staff"  // Operational staff – day-to-day access
	RoleMember ProfileRole = "member" // Regular member – limited read access
)

// validRoles is the single source of truth for allowed roles.
var validRoles = map[ProfileRole]bool{
	RoleOwner:  true,
	RoleAdmin:  true,
	RoleStaff:  true,
	RoleMember: true,
}

// IsValidRole returns true if the given string is a recognised ProfileRole.
func IsValidRole(r string) bool {
	return validRoles[ProfileRole(r)]
}

// ProfileService defines the use cases for managing user profiles.
type ProfileService interface {
	RegisterNewProfile(ctx context.Context, input RegisterInput) (*repo.Profile, error)
}

// RegisterInput represents the Data Transfer Object for creating a new user.
type RegisterInput struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	// Role must be one of: owner, admin, staff, member. Defaults to "member" if empty.
	Role string `json:"role"`
}

type profileService struct {
	repo         repo.ProfileRepo
	authProvider application.AuthProvider
}

// NewProfileService requires both the Repository (for DB) and AuthProvider (for Supabase).
func NewProfileService(r repo.ProfileRepo, auth application.AuthProvider) ProfileService {
	return &profileService{
		repo:         r,
		authProvider: auth,
	}
}

func (s *profileService) RegisterNewProfile(ctx context.Context, input RegisterInput) (*repo.Profile, error) {
	// 1. Validate Input (Basic business rules)
	if input.TenantID == "" || input.Email == "" || input.Password == "" || input.FullName == "" {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.Register", "tenant_id, email, password, and full name are required")
	}

	if input.Role == "" {
		input.Role = string(RoleMember) // default role
	} else if !IsValidRole(input.Role) {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.Register",
			fmt.Sprintf("invalid role %q: must be one of owner, admin, staff, member", input.Role))
	}

	// 2. Call Supabase Admin API to create the Identity (Auth Layer)
	// Pass tenant_id in metadata so the DB trigger `on_auth_user_created`
	// can set it automatically on the profiles row it creates.
	metadata := map[string]interface{}{
		"full_name": input.FullName,
		"role":      input.Role,
		"tenant_id": input.TenantID,
	}

	uid, err := s.authProvider.AdminCreateUser(ctx, input.Email, input.Password, metadata)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ProfileService.Register", "Failed to create identity in Supabase Auth")
	}

	// 3. Return the profile constructed from inputs.
	// The DB trigger `on_auth_user_created` handles the INSERT into public.profiles.
	profile := &repo.Profile{
		ID:       uid,
		TenantID: input.TenantID,
		Email:    input.Email,
		FullName: input.FullName,
		Role:     input.Role,
	}

	return profile, nil
}
