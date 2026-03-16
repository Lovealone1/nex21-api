package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/profiles/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/profiles/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// ProfileListResponse is the concrete paginated response type for the profiles list endpoint.
// Swag does not support generic type annotations, so we declare this concrete alias.
type ProfileListResponse struct {
	Items      []repo.Profile `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int64          `json:"total_pages"`
}

// ProfileHandler takes the profile service and exposes it via REST endpoints.
type ProfileHandler struct {
	profService service.ProfileService
}

// NewProfileHandler creates the handler struct with its dependencies.
func NewProfileHandler(svc service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profService: svc,
	}
}

// RegisterRoutes maps the actual HTTP URLs to handler methods.
// We expect these routes to be mounted under the admin isolated route.
func (h *ProfileHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.RegisterProfile)
	r.Delete("/{id}", h.DeleteProfile)
	r.Patch("/{id}", h.UpdateProfile)
	r.Patch("/{id}/role", h.ChangeRole)
	r.Patch("/{id}/status", h.ToggleStatus)
	r.Get("/", h.ListProfiles)
	r.Get("/{id}", h.GetProfileByID)
}

// RegisterProfile
// @Summary      Create a new User Profile
// @Description  Creates a new identity in Supabase Auth and registers the isolated profile to the active Tenant.
// @Tags         Profiles
// @Accept       json
// @Produce      json
// @Param        request  body      service.RegisterInput  true  "Profile Registration Data"
// @Success      201      {object}  repo.Profile
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/profiles [post]
func (h *ProfileHandler) RegisterProfile(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ProfileHandler.RegisterProfile", "Invalid JSON body format"))
		return
	}

	// We pass the context which already holds the Actor/Tenant identity injected by the middleware
	profile, err := h.profService.RegisterNewProfile(r.Context(), req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(profile)
}

// GetProfileByID
// @Summary      Get a User Profile by ID
// @Description  Fetches a single profile by its Supabase Auth UID. Returns 404 if the profile does not exist.
// @Tags         Profiles
// @Produce      json
// @Param        id   path      string  true  "Profile UUID (Supabase Auth UID)"
// @Success      200  {object}  repo.Profile
// @Failure      400  {object}  errors.HTTPErrorResponse
// @Failure      401  {object}  errors.HTTPErrorResponse
// @Failure      404  {object}  errors.HTTPErrorResponse
// @Failure      500  {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/profiles/{id} [get]
func (h *ProfileHandler) GetProfileByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	profile, err := h.profService.GetProfileByID(r.Context(), id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(profile)
}

// DeleteProfile
// @Summary      Delete a User Profile
// @Description  Permanently removes the user identity from Supabase Auth and deletes their profile record from the database. This action is irreversible.
// @Tags         Profiles
// @Produce      json
// @Param        id   path      string  true  "Profile UUID (Supabase Auth UID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  errors.HTTPErrorResponse
// @Failure      401  {object}  errors.HTTPErrorResponse
// @Failure      404  {object}  errors.HTTPErrorResponse
// @Failure      500  {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/profiles/{id} [delete]
func (h *ProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.profService.DeleteProfile(r.Context(), id); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "User deleted successfully",
	})
}

// UpdateProfile
// @Summary      Partially update a User Profile
// @Description  Applies a partial patch to the profile's mutable fields (full_name and/or email). Only the fields present in the body are updated; omitted fields remain unchanged. If email is changed it is also updated in Supabase Auth.
// @Tags         Profiles
// @Accept       json
// @Produce      json
// @Param        id       path      string              true  "Profile UUID (Supabase Auth UID)"
// @Param        request  body      service.UpdateInput  true  "Fields to update (at least one required)"
// @Success      200      {object}  repo.Profile
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/profiles/{id} [patch]
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req service.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ProfileHandler.UpdateProfile", "Invalid JSON body format"))
		return
	}

	profile, err := h.profService.UpdateProfile(r.Context(), id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(profile)
}

// ChangeRole
// @Summary      Change the role of a User Profile
// @Description  Replaces the current role of the profile with the one specified in the body. Valid roles: owner, admin, staff, member.
// @Tags         Profiles
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "Profile UUID (Supabase Auth UID)"
// @Param        request  body      service.ChangeRoleInput   true  "New Role"
// @Success      200      {object}  repo.Profile
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/profiles/{id}/role [patch]
func (h *ProfileHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req service.ChangeRoleInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ProfileHandler.ChangeRole", "Invalid JSON body format"))
		return
	}

	profile, err := h.profService.ChangeRole(r.Context(), id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(profile)
}

// ToggleStatus
// @Summary      Toggle the active status of a User Profile
// @Description  Inverts the is_active flag of the profile. If the profile is currently active it will be deactivated, and vice-versa. Returns the updated profile with the new status.
// @Tags         Profiles
// @Produce      json
// @Param        id   path      string  true  "Profile UUID (Supabase Auth UID)"
// @Success      200  {object}  repo.Profile
// @Failure      400  {object}  errors.HTTPErrorResponse
// @Failure      401  {object}  errors.HTTPErrorResponse
// @Failure      404  {object}  errors.HTTPErrorResponse
// @Failure      500  {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/profiles/{id}/status [patch]
func (h *ProfileHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	profile, err := h.profService.ToggleStatus(r.Context(), id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(profile)
}

// ListProfiles
// @Summary      List all User Profiles (paginated)
// @Description  Returns a paginated list of all profiles across all tenants. Supports page, limit, sort_by and sort_dir query parameters.
// @Tags         Profiles
// @Produce      json
// @Param        page      query     int     false  "Page number (1-based, default: 1)"
// @Param        limit     query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by   query     string  false  "Sort column: created_at | updated_at | email | full_name | role | is_active"
// @Param        sort_dir  query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200       {object}  ProfileListResponse
// @Failure      401       {object}  errors.HTTPErrorResponse
// @Failure      500       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/profiles [get]
func (h *ProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	// The handler owns the allowlist — the pagination package is domain-agnostic.
	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "email", "full_name", "role", "is_active",
	)

	result, err := h.profService.ListProfiles(r.Context(), page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
