package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/tenant/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/tenant/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// TenantListResponse is the concrete paginated response for the tenants list endpoint.
// Declared explicitly because Swagger does not support generic type annotations.
type TenantListResponse struct {
	Items      []repo.Tenant `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int64         `json:"total_pages"`
}

// TenantHandler exposes the TenantService via REST endpoints.
type TenantHandler struct {
	svc service.TenantService
}

// NewTenantHandler creates the handler with its dependencies wired.
func NewTenantHandler(svc service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

// RegisterRoutes maps HTTP URLs to handler methods.
// Mount under /api/admin/v1/tenants.
func (h *TenantHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateTenant)
	r.Get("/", h.ListTenants)
	r.Get("/{id}", h.GetTenantByID)
	r.Patch("/{id}", h.UpdateTenant)
	r.Delete("/{id}", h.DeleteTenant)
}

// CreateTenant
// @Summary      Create a new Tenant
// @Description  Registers a new tenant. Name and slug are required. Plan defaults to "free", is_active defaults to "true".
// @Tags         Tenants
// @Accept       json
// @Produce      json
// @Param        request  body      service.CreateTenantInput  true  "Tenant creation data"
// @Success      201      {object}  repo.Tenant
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants [post]
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTenantInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "TenantHandler.Create", "Invalid JSON body format"))
		return
	}

	t, err := h.svc.CreateTenant(r.Context(), req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

// GetTenantByID
// @Summary      Get a Tenant by ID
// @Description  Fetches a single tenant by its UUID. Returns 404 if not found.
// @Tags         Tenants
// @Produce      json
// @Param        id   path      string  true  "Tenant UUID"
// @Success      200  {object}  repo.Tenant
// @Failure      400  {object}  errors.HTTPErrorResponse
// @Failure      401  {object}  errors.HTTPErrorResponse
// @Failure      404  {object}  errors.HTTPErrorResponse
// @Failure      500  {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id} [get]
func (h *TenantHandler) GetTenantByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	t, err := h.svc.GetTenantByID(r.Context(), id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(t)
}

// UpdateTenant
// @Summary      Partially update a Tenant
// @Description  Applies a partial patch to mutable tenant fields. Only supplied fields are updated.
// @Tags         Tenants
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "Tenant UUID"
// @Param        request  body      service.UpdateTenantInput  true  "Fields to update (at least one required)"
// @Success      200      {object}  repo.Tenant
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id} [patch]
func (h *TenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req service.UpdateTenantInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "TenantHandler.Update", "Invalid JSON body format"))
		return
	}

	t, err := h.svc.UpdateTenant(r.Context(), id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(t)
}

// DeleteTenant
// @Summary      Delete a Tenant
// @Description  Permanently removes a tenant and all associated data (cascades). This action is irreversible.
// @Tags         Tenants
// @Produce      json
// @Param        id   path      string  true  "Tenant UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  errors.HTTPErrorResponse
// @Failure      401  {object}  errors.HTTPErrorResponse
// @Failure      404  {object}  errors.HTTPErrorResponse
// @Failure      500  {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id} [delete]
func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.DeleteTenant(r.Context(), id); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Tenant deleted successfully",
	})
}

// ListTenants
// @Summary      List all Tenants (paginated)
// @Description  Returns a paginated list of all tenants. Supports page, limit, sort_by and sort_dir query parameters.
// @Tags         Tenants
// @Produce      json
// @Param        page      query     int     false  "Page number (1-based, default: 1)"
// @Param        limit     query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by   query     string  false  "Sort column: created_at | updated_at | name | slug | plan | is_active"
// @Param        sort_dir  query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200       {object}  TenantListResponse
// @Failure      401       {object}  errors.HTTPErrorResponse
// @Failure      500       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants [get]
func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "name", "slug", "plan", "is_active",
	)

	result, err := h.svc.ListTenants(r.Context(), page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
