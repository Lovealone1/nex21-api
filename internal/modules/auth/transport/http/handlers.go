package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/auth/application"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
)

// AuthHandler takes the application service and exposes it via REST.
type AuthHandler struct {
	authService *application.AuthService
}

// NewAuthHandler creates the handler struct with its dependencies.
func NewAuthHandler(service *application.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: service,
	}
}

// RegisterRoutes links standard HTTP methods to the Handler functions.
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.Login)
	r.Get("/me", h.Me)
}

// Login
// @Summary Perform user authentication
// @Description Logs in a user via their email and password returning a Supabase JWT and refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body github.com/Lovealone1/nex21-api/internal/modules/auth/application.LoginRequest true "Credentials"
// @Success 200 {object} github.com/Lovealone1/nex21-api/internal/modules/auth/application.LoginResponse
// @Failure 400 {object} github.com/Lovealone1/nex21-api/internal/platform/apperrors.HTTPErrorResponse
// @Failure 401 {object} github.com/Lovealone1/nex21-api/internal/platform/apperrors.HTTPErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req application.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "AuthHandler.Login", "Invalid JSON body format"))
		return
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Me
// @Summary Get current user profile
// @Description Validates the JWT access token and returns the user context from Supabase GoTrue
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} github.com/Lovealone1/nex21-api/internal/modules/auth/application.UserDTO
// @Failure 401 {object} github.com/Lovealone1/nex21-api/internal/platform/apperrors.HTTPErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		errors.WriteHTTPError(w, errors.New(errors.Unauthenticated, "AuthHandler.Me", "Missing Authorization header"))
		return
	}

	token := authHeader
	// Automatically strip "Bearer " prefix if provided, otherwise assume the raw token was passed (common in Swagger UI)
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	user, err := h.authService.Me(r.Context(), token)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}
