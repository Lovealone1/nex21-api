package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	_ "github.com/Lovealone1/nex21-api/internal/modules/profiles/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/profiles/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
)

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
// We expect these routes to be mounted under a Tenant-Scoped protected route.
func (h *ProfileHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.RegisterProfile)
}

// RegisterProfile
// @Summary Create a new User Profile
// @Description Creates a new identity in Supabase Auth and registers the isolated profile to the active Tenant.
// @Tags Profiles
// @Accept json
// @Produce json
// @Param request body service.RegisterInput true "Profile Registration Data"
// @Success 201 {object} repo.Profile
// @Failure 400 {object} errors.HTTPErrorResponse
// @Failure 401 {object} errors.HTTPErrorResponse
// @Failure 500 {object} errors.HTTPErrorResponse
// @Security BearerAuth
// @Router /api/admin/v1/profiles [post]
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
