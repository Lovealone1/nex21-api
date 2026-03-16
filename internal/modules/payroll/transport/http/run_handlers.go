package payrollhttp

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	payrollservice "github.com/Lovealone1/nex21-api/internal/modules/payroll/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

type RunListResponse struct {
	Items      []payrollrepo.PayrollRun `json:"items"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	TotalPages int64                    `json:"total_pages"`
}

type PayrollRunHandler struct {
	svc payrollservice.PayrollRunService
}

func NewPayrollRunHandler(svc payrollservice.PayrollRunService) *PayrollRunHandler {
	return &PayrollRunHandler{svc: svc}
}

func (h *PayrollRunHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateRun)
	r.Get("/", h.ListRuns)
	r.Get("/{runId}", h.GetRunByID)
	r.Patch("/{runId}", h.UpdateRun)
	r.Delete("/{runId}", h.DeleteRun)
	r.Patch("/{runId}/status", h.UpdateRunStatus)
}

// CreateRun
// @Summary      Create Payroll Run
// @Description  Creates a new payroll batch/run. Default status is 'draft'.
// @Tags         Payroll
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                       true  "Tenant UUID"
// @Param        request  body      payrollservice.CreateRunInput true  "Run data"
// @Success      201      {object}  payrollrepo.PayrollRun
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      409      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs [post]
func (h *PayrollRunHandler) CreateRun(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req payrollservice.CreateRunInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "PayrollRunHandler.Create", "Invalid JSON format"))
		return
	}

	run, err := h.svc.CreateRun(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(run)
}

// GetRunByID
// @Summary      Get Payroll Run
// @Description  Retrieves a payroll run by ID.
// @Tags         Payroll
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        runId    path      string  true  "Run UUID"
// @Success      200      {object}  payrollrepo.PayrollRun
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId} [get]
func (h *PayrollRunHandler) GetRunByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")

	run, err := h.svc.GetRunByID(r.Context(), tenantID, runID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(run)
}

// UpdateRun
// @Summary      Update Payroll Run
// @Description  Applies partial update to editable fields of a draft run.
// @Tags         Payroll
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                       true  "Tenant UUID"
// @Param        runId    path      string                       true  "Run UUID"
// @Param        request  body      payrollservice.UpdateRunInput true  "Fields to update"
// @Success      200      {object}  payrollrepo.PayrollRun
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      412      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId} [patch]
func (h *PayrollRunHandler) UpdateRun(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")

	var req payrollservice.UpdateRunInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "PayrollRunHandler.Update", "Invalid JSON format"))
		return
	}

	run, err := h.svc.UpdateRun(r.Context(), tenantID, runID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(run)
}

// UpdateRunStatus
// @Summary      Update Payroll Run Status
// @Description  Transitions a payroll run between statuses.
// @Tags         Payroll
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                             true  "Tenant UUID"
// @Param        runId    path      string                             true  "Run UUID"
// @Param        request  body      payrollservice.UpdateRunStatusInput true  "Status Data"
// @Success      200      {object}  payrollrepo.PayrollRun
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      412      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId}/status [patch]
func (h *PayrollRunHandler) UpdateRunStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")

	var req payrollservice.UpdateRunStatusInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "PayrollRunHandler.UpdateStatus", "Invalid JSON format"))
		return
	}

	run, err := h.svc.UpdateRunStatus(r.Context(), tenantID, runID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(run)
}

// DeleteRun
// @Summary      Delete Payroll Run
// @Description  Permanently removes a draft payroll run.
// @Tags         Payroll
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        runId    path      string  true  "Run UUID"
// @Success      200      {object}  map[string]string
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      412      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId} [delete]
func (h *PayrollRunHandler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")

	if err := h.svc.DeleteRun(r.Context(), tenantID, runID); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Payroll run deleted successfully",
	})
}

// ListRuns
// @Summary      List Payroll Runs
// @Description  Returns a paginated list of payroll runs with optional filters.
// @Tags         Payroll
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        status       query     string  false  "Filter by status"
// @Param        frequency    query     string  false  "Filter by frequency"
// @Param        location_id  query     string  false  "Filter by location"
// @Param        period_start query     string  false  "Filter period_start >="
// @Param        period_end   query     string  false  "Filter period_end <="
// @Param        sort_by      query     string  false  "Sort column: created_at | period_start"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC"
// @Success      200          {object}  RunListResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs [get]
func (h *PayrollRunHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "period_start", "period_end", "pay_date", "total", "status",
	)

	filters := payrollrepo.RunFilters{}
	if s := r.URL.Query().Get("status"); s != "" {
		st := payrolldomain.RunStatus(s)
		filters.Status = &st
	}
	if f := r.URL.Query().Get("frequency"); f != "" {
		fr := payrolldomain.RunFrequency(f)
		filters.Frequency = &fr
	}
	if lid := r.URL.Query().Get("location_id"); lid != "" {
		filters.LocationID = &lid
	}
	if ps := r.URL.Query().Get("period_start"); ps != "" {
		if t, err := time.Parse("2006-01-02", ps); err == nil {
			filters.PeriodStart = &t
		}
	}
	if pe := r.URL.Query().Get("period_end"); pe != "" {
		if t, err := time.Parse("2006-01-02", pe); err == nil {
			filters.PeriodEnd = &t
		}
	}

	result, err := h.svc.ListRuns(r.Context(), tenantID, filters, page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
