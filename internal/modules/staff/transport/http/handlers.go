package staffhttp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	staffrepo "github.com/Lovealone1/nex21-api/internal/modules/staff/repo"
	staffservice "github.com/Lovealone1/nex21-api/internal/modules/staff/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// StaffListResponse represents a paginated list of staff members
type StaffListResponse struct {
	Items      []staffrepo.Staff `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int64             `json:"total_pages"`
}

type StaffHandler struct {
	svc staffservice.StaffService
}

func NewStaffHandler(svc staffservice.StaffService) *StaffHandler {
	return &StaffHandler{svc: svc}
}

// RegisterRoutes sets up the REST endpoints for staff.
// Mount under: /api/admin/v1/tenants/{tenantId}/staff
func (h *StaffHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateStaff)
	r.Get("/", h.ListStaff)
	r.Get("/{id}", h.GetStaffByID)
	r.Patch("/{id}", h.UpdateStaff)
	r.Delete("/{id}", h.DeleteStaff)
	r.Patch("/{id}/status", h.ToggleStatus)
	r.Patch("/{id}/role", h.ChangeRole)
}

// CreateStaff
// @Summary      Create/Hire a new Staff Member
// @Description  Creates a new staff record in the specified tenant workspace.
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                   true  "Tenant UUID"
// @Param        request  body      staffservice.CreateStaffInput true  "Staff member data"
// @Success      201      {object}  staffrepo.Staff
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff [post]
func (h *StaffHandler) CreateStaff(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req staffservice.CreateStaffInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "StaffHandler.Create", "Invalid JSON format"))
		return
	}

	st, err := h.svc.CreateStaff(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(st)
}

// GetStaffByID
// @Summary      Get a Staff Member by ID
// @Description  Returns the staff details if found inside the tenant.
// @Tags         Staff
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Staff UUID"
// @Success      200      {object}  staffrepo.Staff
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{id} [get]
func (h *StaffHandler) GetStaffByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	st, err := h.svc.GetStaffByID(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(st)
}

// UpdateStaff
// @Summary      Update Staff Information
// @Description  Applies a partial update to a staff member (display name, contact, etc).
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                   true  "Tenant UUID"
// @Param        id       path      string                   true  "Staff UUID"
// @Param        request  body      staffservice.UpdateStaffInput true  "Fields to update"
// @Success      200      {object}  staffrepo.Staff
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{id} [patch]
func (h *StaffHandler) UpdateStaff(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req staffservice.UpdateStaffInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "StaffHandler.Update", "Invalid JSON format"))
		return
	}

	st, err := h.svc.UpdateStaff(r.Context(), tenantID, id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(st)
}

// DeleteStaff
// @Summary      Delete a Staff Member
// @Description  Permanently removes a staff record from the tenant.
// @Tags         Staff
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Staff UUID"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{id} [delete]
func (h *StaffHandler) DeleteStaff(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	if err := h.svc.DeleteStaff(r.Context(), tenantID, id); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Staff member deleted successfully",
	})
}

// ListStaff
// @Summary      List all Staff Members for a Tenant
// @Description  Returns a paginated list of staff members.
// @Tags         Staff
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by      query     string  false  "Sort column: created_at | display_name | staff_role | is_active"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200          {object}  StaffListResponse
// @Failure      400          {object}  errors.HTTPErrorResponse
// @Failure      500          {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff [get]
func (h *StaffHandler) ListStaff(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "display_name", "staff_role", "is_active",
	)

	result, err := h.svc.ListStaff(r.Context(), tenantID, page)
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
// @Summary      Toggle Staff Status
// @Description  Inverts a staff member's is_active status.
// @Tags         Staff
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Staff UUID"
// @Success      200      {object}  staffrepo.Staff
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{id}/status [patch]
func (h *StaffHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	st, err := h.svc.ToggleStatus(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(st)
}

// ChangeRole
// @Summary      Change Staff Role
// @Description  Replaces a staff member's role with a new one from the allowed ENUM.
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                   true  "Tenant UUID"
// @Param        id       path      string                   true  "Staff UUID"
// @Param        request  body      staffservice.ChangeRoleInput  true  "New Role Data"
// @Success      200      {object}  staffrepo.Staff
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{id}/role [patch]
func (h *StaffHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req staffservice.ChangeRoleInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "StaffHandler.ChangeRole", "Invalid JSON format"))
		return
	}

	st, err := h.svc.ChangeRole(r.Context(), tenantID, id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(st)
}
