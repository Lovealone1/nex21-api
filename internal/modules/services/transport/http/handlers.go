package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/services/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/services/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// ServiceListResponse represents a paginated list of services
type ServiceListResponse struct {
	Items      []repo.Service `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int64          `json:"total_pages"`
}

type ServiceHandler struct {
	svc service.ServiceService
}

func NewServiceHandler(svc service.ServiceService) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

// RegisterRoutes sets up the REST endpoints for services.
// This is typically mounted onto the router UNDER `/api/admin/v1/tenants/{tenantId}/services`
func (h *ServiceHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateService)
	r.Get("/", h.ListServices)
	r.Get("/{id}", h.GetServiceByID)
	r.Patch("/{id}", h.UpdateService)
	r.Delete("/{id}", h.DeleteService)
	r.Patch("/{id}/status", h.ToggleStatus)
}

// CreateService
// @Summary      Create a new Service
// @Description  Creates a new service in the specified tenant workspace.
// @Tags         Services
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                     true  "Tenant UUID"
// @Param        request  body      service.CreateServiceInput true  "Service data"
// @Success      201      {object}  repo.Service
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/services [post]
func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req service.CreateServiceInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ServiceHandler.Create", "Invalid JSON format"))
		return
	}

	svc, err := h.svc.CreateService(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(svc)
}

// GetServiceByID
// @Summary      Get a Service by ID
// @Description  Returns the service details if found inside the tenant.
// @Tags         Services
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Service UUID"
// @Success      200      {object}  repo.Service
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/services/{id} [get]
func (h *ServiceHandler) GetServiceByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	svc, err := h.svc.GetServiceByID(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(svc)
}

// UpdateService
// @Summary      Update a Service
// @Description  Applies a partial update to a service. Only non-null fields will be modified.
// @Tags         Services
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                     true  "Tenant UUID"
// @Param        id       path      string                     true  "Service UUID"
// @Param        request  body      service.UpdateServiceInput true  "Fields to update"
// @Success      200      {object}  repo.Service
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/services/{id} [patch]
func (h *ServiceHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req service.UpdateServiceInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ServiceHandler.Update", "Invalid JSON format"))
		return
	}

	svc, err := h.svc.UpdateService(r.Context(), tenantID, id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(svc)
}

// DeleteService
// @Summary      Delete a Service
// @Description  Permanently deletes a service from the tenant.
// @Tags         Services
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Service UUID"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/services/{id} [delete]
func (h *ServiceHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	if err := h.svc.DeleteService(r.Context(), tenantID, id); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Service deleted successfully",
	})
}

// ListServices
// @Summary      List all Services for a Tenant
// @Description  Returns a paginated list of services.
// @Tags         Services
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by      query     string  false  "Sort column: created_at | name | price | category | is_active | sku | duration_minutes"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200          {object}  ServiceListResponse
// @Failure      400          {object}  errors.HTTPErrorResponse
// @Failure      401          {object}  errors.HTTPErrorResponse
// @Failure      500          {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/services [get]
func (h *ServiceHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "name", "price", "category", "is_active", "sku", "duration_minutes",
	)

	result, err := h.svc.ListServices(r.Context(), tenantID, page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// ToggleStatus
// @Summary      Toggle Service Status
// @Description  Automatically flips a service's is_active status boolean.
// @Tags         Services
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Service UUID"
// @Success      200      {object}  repo.Service
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/services/{id}/status [patch]
func (h *ServiceHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	svc, err := h.svc.ToggleStatus(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(svc)
}
