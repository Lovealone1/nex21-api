package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/locations/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/locations/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// LocationListResponse represents a paginated list of locations
type LocationListResponse struct {
	Items      []repo.Location `json:"items"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int64           `json:"total_pages"`
}

type LocationHandler struct {
	svc service.LocationService
}

func NewLocationHandler(svc service.LocationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

// RegisterRoutes sets up the REST endpoints for locations.
// This is typically mounted onto the router UNDER `/api/admin/v1/tenants/{tenantId}/locations`
func (h *LocationHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateLocation)
	r.Get("/", h.ListLocations)
	r.Get("/{id}", h.GetLocationByID)
	r.Patch("/{id}", h.UpdateLocation)
	r.Delete("/{id}", h.DeleteLocation)
	r.Patch("/{id}/status", h.ToggleStatus)
}

// CreateLocation
// @Summary      Create a new Location
// @Description  Creates a new location in the specified tenant workspace.
// @Tags         Locations
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                      true  "Tenant UUID"
// @Param        request  body      service.CreateLocationInput true  "Location data"
// @Success      201      {object}  repo.Location
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/locations [post]
func (h *LocationHandler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req service.CreateLocationInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "LocationHandler.Create", "Invalid JSON format"))
		return
	}

	l, err := h.svc.CreateLocation(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(l)
}

// GetLocationByID
// @Summary      Get a Location by ID
// @Description  Returns the location details if found inside the tenant.
// @Tags         Locations
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Location UUID"
// @Success      200      {object}  repo.Location
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/locations/{id} [get]
func (h *LocationHandler) GetLocationByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	l, err := h.svc.GetLocationByID(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(l)
}

// UpdateLocation
// @Summary      Update a Location
// @Description  Applies a partial update to a location. Only non-null fields will be modified.
// @Tags         Locations
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                      true  "Tenant UUID"
// @Param        id       path      string                      true  "Location UUID"
// @Param        request  body      service.UpdateLocationInput true  "Fields to update"
// @Success      200      {object}  repo.Location
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/locations/{id} [patch]
func (h *LocationHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req service.UpdateLocationInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "LocationHandler.Update", "Invalid JSON format"))
		return
	}

	l, err := h.svc.UpdateLocation(r.Context(), tenantID, id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(l)
}

// DeleteLocation
// @Summary      Delete a Location
// @Description  Permanently deletes a location from the tenant.
// @Tags         Locations
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Location UUID"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/locations/{id} [delete]
func (h *LocationHandler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	if err := h.svc.DeleteLocation(r.Context(), tenantID, id); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Location deleted successfully",
	})
}

// ListLocations
// @Summary      List all Locations for a Tenant
// @Description  Returns a paginated list of locations.
// @Tags         Locations
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by      query     string  false  "Sort column: created_at | name | code | is_active"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200          {object}  LocationListResponse
// @Failure      400          {object}  errors.HTTPErrorResponse
// @Failure      401          {object}  errors.HTTPErrorResponse
// @Failure      500          {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/locations [get]
func (h *LocationHandler) ListLocations(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "name", "code", "is_active",
	)

	result, err := h.svc.ListLocations(r.Context(), tenantID, page)
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
// @Summary      Toggle Location Status
// @Description  Automatically flips a location's is_active status boolean.
// @Tags         Locations
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Location UUID"
// @Success      200      {object}  repo.Location
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/locations/{id}/status [patch]
func (h *LocationHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	l, err := h.svc.ToggleStatus(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(l)
}
