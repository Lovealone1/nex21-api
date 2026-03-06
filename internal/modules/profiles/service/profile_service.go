package service

import (
	"context"
	"fmt"

	"github.com/Lovealone1/nex21-api/internal/core/store"
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

// ─── DTOs ────────────────────────────────────────────────────────────────────

// RegisterInput represents the Data Transfer Object for creating a new user.
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	// Role must be one of: owner, admin, staff, member. Defaults to "member" if empty.
	Role string `json:"role"`
}

// UpdateInput holds the mutable fields for a partial profile patch.
// Only non-empty/non-nil fields are applied.
type UpdateInput struct {
	// FullName is the new display name (optional).
	FullName *string `json:"full_name,omitempty"`
	// Email is the new email; also updated in Supabase Auth (optional).
	Email *string `json:"email,omitempty"`
}

// ChangeRoleInput holds the new role to assign.
type ChangeRoleInput struct {
	// Role must be one of: owner, admin, staff, member.
	Role string `json:"role"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

// ProfileService defines the use cases for managing user profiles.
type ProfileService interface {
	RegisterNewProfile(ctx context.Context, input RegisterInput) (*repo.Profile, error)
	// DeleteProfile removes the user identity from Supabase Auth and their profile row.
	DeleteProfile(ctx context.Context, id string) error
	// UpdateProfile applies a partial patch (full_name, email) to the profile.
	UpdateProfile(ctx context.Context, id string, input UpdateInput) (*repo.Profile, error)
	// ChangeRole replaces the current role of a profile with the one provided.
	ChangeRole(ctx context.Context, id string, input ChangeRoleInput) (*repo.Profile, error)
	// ToggleStatus inverts the is_active flag of the profile.
	ToggleStatus(ctx context.Context, id string) (*repo.Profile, error)
	// ListProfiles returns a paginated list of all profiles.
	ListProfiles(ctx context.Context, page store.Page) (store.ResultList[repo.Profile], error)
	// GetProfileByID fetches a single profile by its UID.
	GetProfileByID(ctx context.Context, id string) (*repo.Profile, error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

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
	if input.Email == "" || input.Password == "" || input.FullName == "" {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.Register", "email, password, and full name are required")
	}

	if input.Role == "" {
		input.Role = string(RoleMember) // default role
	} else if !IsValidRole(input.Role) {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.Register",
			fmt.Sprintf("invalid role %q: must be one of owner, admin, staff, member", input.Role))
	}

	// 2. Call Supabase Admin API to create the Identity (Auth Layer)
	// The DB trigger `on_auth_user_created`
	// can handle profiles row automatically, with tenant_id being NULL.
	metadata := map[string]interface{}{
		"full_name": input.FullName,
		"role":      input.Role,
	}

	uid, err := s.authProvider.AdminCreateUser(ctx, input.Email, input.Password, metadata)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ProfileService.Register", "Failed to create identity in Supabase Auth")
	}

	// 3. Return the profile constructed from inputs.
	// The DB trigger `on_auth_user_created` handles the INSERT into public.profiles.
	profile := &repo.Profile{
		ID:       uid,
		TenantID: nil, // Starts without a tenant
		Email:    input.Email,
		FullName: input.FullName,
		Role:     input.Role,
		IsActive: true,
	}

	return profile, nil
}

// DeleteProfile removes a user from Supabase Auth.
// The FK `profiles(id) REFERENCES auth.users(id) ON DELETE CASCADE` guarantees
// that the profile row is automatically removed by PostgreSQL the instant Supabase
// Auth deletes the auth.users record. Calling repo.Delete() explicitly afterwards
// would fail because the tenant session can no longer validate the (already gone)
// profile membership.
func (s *profileService) DeleteProfile(ctx context.Context, id string) error {
	if id == "" {
		return errors.New(errors.InvalidArgument, "ProfileService.Delete", "id is required")
	}

	// Deleting the auth.users row triggers the CASCADE that removes the profile row.
	if err := s.authProvider.AdminDeleteUser(ctx, id); err != nil {
		return errors.Wrap(err, errors.Internal, "ProfileService.Delete", "Failed to delete identity in Supabase Auth")
	}

	return nil
}

// UpdateProfile applies a partial update on the mutable fields: full_name and/or email.
// If email changes it is also updated in Supabase Auth.
func (s *profileService) UpdateProfile(ctx context.Context, id string, input UpdateInput) (*repo.Profile, error) {
	if id == "" {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.Update", "id is required")
	}
	if input.FullName == nil && input.Email == nil {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.Update", "at least one of full_name or email must be provided")
	}

	// 1. If email is changing, update auth identity first.
	if input.Email != nil {
		authUpdates := map[string]interface{}{"email": *input.Email}
		if input.FullName != nil {
			authUpdates["full_name"] = *input.FullName
		}
		if err := s.authProvider.AdminUpdateUser(ctx, id, authUpdates); err != nil {
			return nil, errors.Wrap(err, errors.Internal, "ProfileService.Update", "Failed to update identity in Supabase Auth")
		}
	}

	// 2. Persist changes in DB via admin (no tenant session required on admin routes).
	updated, err := s.repo.AdminUpdate(ctx, id, repo.UpdateFields{
		FullName: input.FullName,
		Email:    input.Email,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ProfileService.Update", "Failed to update profile in database")
	}

	return updated, nil
}

// ChangeRole replaces the profile's role with the one provided in the input.
func (s *profileService) ChangeRole(ctx context.Context, id string, input ChangeRoleInput) (*repo.Profile, error) {
	if id == "" {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.ChangeRole", "id is required")
	}
	if !IsValidRole(input.Role) {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.ChangeRole",
			fmt.Sprintf("invalid role %q: must be one of owner, admin, staff, member", input.Role))
	}

	updated, err := s.repo.AdminUpdate(ctx, id, repo.UpdateFields{
		Role: &input.Role,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ProfileService.ChangeRole", "Failed to update role in database")
	}

	return updated, nil
}

// ToggleStatus atomically inverts the is_active flag of the profile.
// Uses a single UPDATE query to avoid pgx prepared-statement conflicts (42P05)
// that occur when two queries share the same pool connection.
func (s *profileService) ToggleStatus(ctx context.Context, id string) (*repo.Profile, error) {
	if id == "" {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.ToggleStatus", "id is required")
	}

	updated, err := s.repo.AdminToggleStatus(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ProfileService.ToggleStatus", "Failed to toggle status in database")
	}

	return updated, nil
}

// ListProfiles returns a paginated list of all profiles.
func (s *profileService) ListProfiles(ctx context.Context, page store.Page) (store.ResultList[repo.Profile], error) {
	result, err := s.repo.AdminListAll(ctx, page)
	if err != nil {
		return store.ResultList[repo.Profile]{}, errors.Wrap(err, errors.Internal, "ProfileService.ListProfiles", "Failed to list profiles")
	}

	return result, nil
}

// GetProfileByID fetches a single profile by its UID.
// Uses AdminGetByID since this is called from an admin route with no tenant session.
func (s *profileService) GetProfileByID(ctx context.Context, id string) (*repo.Profile, error) {
	if id == "" {
		return nil, errors.New(errors.InvalidArgument, "ProfileService.GetProfileByID", "id is required")
	}

	profile, err := s.repo.AdminGetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.NotFound, "ProfileService.GetProfileByID", "Profile not found")
	}

	return profile, nil
}
