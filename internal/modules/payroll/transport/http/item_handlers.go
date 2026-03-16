package payrollhttp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	payrollservice "github.com/Lovealone1/nex21-api/internal/modules/payroll/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

type ItemListResponse struct {
	Items      []payrollrepo.PayrollItem `json:"items"`
	Total      int64                     `json:"total"`
	Page       int                       `json:"page"`
	Limit      int                       `json:"limit"`
	TotalPages int64                     `json:"total_pages"`
}

type PayrollItemHandler struct {
	svc payrollservice.PayrollItemService
}

func NewPayrollItemHandler(svc payrollservice.PayrollItemService) *PayrollItemHandler {
	return &PayrollItemHandler{svc: svc}
}

func (h *PayrollItemHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateItem)
	r.Get("/", h.ListItems)
	r.Get("/{itemId}", h.GetItemByID)
	r.Patch("/{itemId}", h.UpdateItem)
	r.Delete("/{itemId}", h.DeleteItem)
}

// CreateItem
// @Summary      Create Payroll Item
// @Description  Creates a new line item in a draft payroll run.
// @Tags         Payroll
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                        true  "Tenant UUID"
// @Param        runId    path      string                        true  "Run UUID"
// @Param        request  body      payrollservice.CreateItemInput true  "Item data"
// @Success      201      {object}  payrollrepo.PayrollItem
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      409      {object}  errors.HTTPErrorResponse
// @Failure      412      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId}/items [post]
func (h *PayrollItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")

	var req payrollservice.CreateItemInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "PayrollItemHandler.Create", "Invalid JSON format"))
		return
	}

	item, err := h.svc.CreateItem(r.Context(), tenantID, runID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

// GetItemByID
// @Summary      Get Payroll Item
// @Description  Retrieves a specific payroll item.
// @Tags         Payroll
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        runId    path      string  true  "Run UUID"
// @Param        itemId   path      string  true  "Item UUID"
// @Success      200      {object}  payrollrepo.PayrollItem
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId}/items/{itemId} [get]
func (h *PayrollItemHandler) GetItemByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")
	itemID := chi.URLParam(r, "itemId")

	item, err := h.svc.GetItemByID(r.Context(), tenantID, runID, itemID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(item)
}

// UpdateItem
// @Summary      Update Payroll Item
// @Description  Applies partial update to an item (if run is draft).
// @Tags         Payroll
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                        true  "Tenant UUID"
// @Param        runId    path      string                        true  "Run UUID"
// @Param        itemId   path      string                        true  "Item UUID"
// @Param        request  body      payrollservice.UpdateItemInput true  "Fields to update"
// @Success      200      {object}  payrollrepo.PayrollItem
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      412      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId}/items/{itemId} [patch]
func (h *PayrollItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")
	itemID := chi.URLParam(r, "itemId")

	var req payrollservice.UpdateItemInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "PayrollItemHandler.Update", "Invalid JSON format"))
		return
	}

	item, err := h.svc.UpdateItem(r.Context(), tenantID, runID, itemID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(item)
}

// DeleteItem
// @Summary      Delete Payroll Item
// @Description  Removes an item from a draft payroll run.
// @Tags         Payroll
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        runId    path      string  true  "Run UUID"
// @Param        itemId   path      string  true  "Item UUID"
// @Success      200      {object}  map[string]string
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      412      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId}/items/{itemId} [delete]
func (h *PayrollItemHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")
	itemID := chi.URLParam(r, "itemId")

	if err := h.svc.DeleteItem(r.Context(), tenantID, runID, itemID); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Payroll item deleted successfully",
	})
}

// ListItems
// @Summary      List Payroll Items
// @Description  Returns a paginated list of items in a run.
// @Tags         Payroll
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        runId        path      string  true   "Run UUID"
// @Param        page         query     int     false  "Page number"
// @Param        limit        query     int     false  "Records per page"
// @Param        staff_id     query     string  false  "Filter by staff"
// @Param        concept      query     string  false  "Filter by concept"
// @Param        line_type    query     string  false  "Filter by type"
// @Param        sort_by      query     string  false  "Sort column"
// @Param        sort_dir     query     string  false  "Sort direction"
// @Success      200          {object}  ItemListResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/payroll/runs/{runId}/items [get]
func (h *PayrollItemHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	runID := chi.URLParam(r, "runId")

	page := pagination.ParseRequest(r, "created_at", "updated_at", "amount", "line_type")

	filters := payrollrepo.ItemFilters{}
	if sid := r.URL.Query().Get("staff_id"); sid != "" {
		filters.StaffID = &sid
	}
	if c := r.URL.Query().Get("concept"); c != "" {
		ic := payrolldomain.ItemConcept(c)
		filters.Concept = &ic
	}
	if l := r.URL.Query().Get("line_type"); l != "" {
		lt := payrolldomain.LineType(l)
		filters.LineType = &lt
	}

	result, err := h.svc.ListItems(r.Context(), tenantID, runID, filters, page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
