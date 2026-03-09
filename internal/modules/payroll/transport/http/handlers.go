package payrollhttp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	payrollservice "github.com/Lovealone1/nex21-api/internal/modules/payroll/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// StaffPayListResponse represents a paginated list of staff pay plans
type StaffPayListResponse struct {
	Items      []payrollrepo.StaffPay `json:"items"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int64                  `json:"total_pages"`
}

type StaffPayHandler struct {
	svc payrollservice.StaffPayService
}

func NewStaffPayHandler(svc payrollservice.StaffPayService) *StaffPayHandler {
	return &StaffPayHandler{svc: svc}
}

// RegisterRoutes sets up the REST endpoints for staff pay.
// Mount under: /api/admin/v1/tenants/{tenantId}/staff/{staffId}/pay
func (h *StaffPayHandler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.GetActiveStaffPay)
	r.Post("/", h.CreateStaffPay)
	r.Patch("/", h.UpdateStaffPay)
	r.Get("/history", h.ListStaffPayHistory)
	r.Patch("/{payId}/status", h.ToggleStaffPayStatus)
}

// GetActiveStaffPay
// @Summary      Get Active Staff Pay
// @Description  Returns the currently active pay plan for a staff member.
// @Tags         Staff Compensation
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        staffId  path      string  true  "Staff UUID"
// @Success      200      {object}  payrollrepo.StaffPay
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{staffId}/pay [get]
func (h *StaffPayHandler) GetActiveStaffPay(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	staffID := chi.URLParam(r, "staffId")

	pay, err := h.svc.GetActiveStaffPay(r.Context(), tenantID, staffID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pay)
}

// CreateStaffPay
// @Summary      Create Staff Pay Plan
// @Description  Creates a new pay plan for a staff member.
// @Tags         Staff Compensation
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                           true  "Tenant UUID"
// @Param        staffId  path      string                           true  "Staff UUID"
// @Param        request  body      payrollservice.CreateStaffPayInput true  "Pay Plan data"
// @Success      201      {object}  payrollrepo.StaffPay
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{staffId}/pay [post]
func (h *StaffPayHandler) CreateStaffPay(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	staffID := chi.URLParam(r, "staffId")

	var req payrollservice.CreateStaffPayInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "StaffPayHandler.Create", "Invalid JSON format"))
		return
	}

	pay, err := h.svc.CreateStaffPay(r.Context(), tenantID, staffID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(pay)
}

// UpdateStaffPay
// @Summary      Update Active Pay Plan
// @Description  Applies a partial update to the active pay plan.
// @Tags         Staff Compensation
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                           true  "Tenant UUID"
// @Param        staffId  path      string                           true  "Staff UUID"
// @Param        request  body      payrollservice.UpdateStaffPayInput true  "Fields to update"
// @Success      200      {object}  payrollrepo.StaffPay
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{staffId}/pay [patch]
func (h *StaffPayHandler) UpdateStaffPay(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	staffID := chi.URLParam(r, "staffId")

	// Get active pay plan ID to update
	activePay, err := h.svc.GetActiveStaffPay(r.Context(), tenantID, staffID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	var req payrollservice.UpdateStaffPayInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "StaffPayHandler.Update", "Invalid JSON format"))
		return
	}

	pay, err := h.svc.UpdateStaffPay(r.Context(), tenantID, staffID, activePay.ID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pay)
}

// ListStaffPayHistory
// @Summary      List Staff Pay History
// @Description  Returns a paginated list of staff pay plans.
// @Tags         Staff Compensation
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        staffId      path      string  true   "Staff UUID"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by      query     string  false  "Sort column: created_at | effective_from | pay_frequency"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200          {object}  StaffPayListResponse
// @Failure      400          {object}  errors.HTTPErrorResponse
// @Failure      500          {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{staffId}/pay/history [get]
func (h *StaffPayHandler) ListStaffPayHistory(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	staffID := chi.URLParam(r, "staffId")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "effective_from", "pay_frequency", "is_active",
	)

	result, err := h.svc.ListStaffPayHistory(r.Context(), tenantID, staffID, page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// ToggleStaffPayStatus
// @Summary      Toggle Staff Pay Plan Status
// @Description  Activates or deactivates a specific staff pay plan.
// @Tags         Staff Compensation
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        staffId  path      string  true  "Staff UUID"
// @Param        payId    path      string  true  "Pay Plan UUID"
// @Success      200      {object}  payrollrepo.StaffPay
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/staff/{staffId}/pay/{payId}/status [patch]
func (h *StaffPayHandler) ToggleStaffPayStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	staffID := chi.URLParam(r, "staffId")
	payID := chi.URLParam(r, "payId")

	pay, err := h.svc.ToggleStaffPayStatus(r.Context(), tenantID, staffID, payID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pay)
}
