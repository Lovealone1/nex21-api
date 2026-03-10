package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/finance/domain"
	"github.com/Lovealone1/nex21-api/internal/modules/finance/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/finance/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

type AccountListResponse struct {
	Items      []repo.Account `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int64          `json:"total_pages"`
}

type AccountHandler struct {
	svc service.AccountService
}

func NewAccountHandler(svc service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateAccount)
	r.Get("/", h.ListAccounts)
	r.Get("/default", h.GetDefaultAccount) // Specific route must come before /{accountId}
	r.Get("/{accountId}", h.GetAccountByID)
	r.Patch("/{accountId}", h.UpdateAccount)
	r.Delete("/{accountId}", h.DeleteAccount)

	// Status actions
	r.Patch("/{accountId}/status", h.ToggleAccountStatus)
	r.Patch("/{accountId}/default", h.SetDefaultAccount)
}

// CreateAccount
// @Summary      Create Account
// @Description  Creates a new financial account for the tenant.
// @Tags         Accounts
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                              true  "Tenant UUID"
// @Param        request  body      service.CreateAccountInput true  "Account data"
// @Success      201      {object}  repo.Account
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      409      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts [post]
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req service.CreateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "AccountHandler.Create", "Invalid JSON format"))
		return
	}

	acc, err := h.svc.CreateAccount(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(acc)
}

// GetAccountByID
// @Summary      Get Account
// @Description  Retrieves an account by its ID.
// @Tags         Accounts
// @Produce      json
// @Param        tenantId  path      string  true  "Tenant UUID"
// @Param        accountId path      string  true  "Account UUID"
// @Success      200       {object}  repo.Account
// @Failure      404       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts/{accountId} [get]
func (h *AccountHandler) GetAccountByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	accountID := chi.URLParam(r, "accountId")

	acc, err := h.svc.GetAccountByID(r.Context(), tenantID, accountID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(acc)
}

// GetDefaultAccount
// @Summary      Get Default Account
// @Description  Fetches the single active default account for a tenant.
// @Tags         Accounts
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Success      200      {object}  repo.Account
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts/default [get]
func (h *AccountHandler) GetDefaultAccount(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	acc, err := h.svc.GetDefaultAccount(r.Context(), tenantID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(acc)
}

// UpdateAccount
// @Summary      Update Account
// @Description  Partially updates an account's details.
// @Tags         Accounts
// @Accept       json
// @Produce      json
// @Param        tenantId  path      string                              true  "Tenant UUID"
// @Param        accountId path      string                              true  "Account UUID"
// @Param        request   body      service.UpdateAccountInput true  "Fields to update"
// @Success      200       {object}  repo.Account
// @Failure      400       {object}  errors.HTTPErrorResponse
// @Failure      409       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts/{accountId} [patch]
func (h *AccountHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	accountID := chi.URLParam(r, "accountId")

	var req service.UpdateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "AccountHandler.Update", "Invalid JSON format"))
		return
	}

	acc, err := h.svc.UpdateAccount(r.Context(), tenantID, accountID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(acc)
}

// DeleteAccount
// @Summary      Delete Account
// @Description  Deletes an account if unused, or soft-deletes it if transactions reference it.
// @Tags         Accounts
// @Produce      json
// @Param        tenantId  path      string  true  "Tenant UUID"
// @Param        accountId path      string  true  "Account UUID"
// @Success      200       {object}  map[string]string
// @Failure      404       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts/{accountId} [delete]
func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	accountID := chi.URLParam(r, "accountId")

	if err := h.svc.DeleteAccount(r.Context(), tenantID, accountID); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Account successfully deleted (or archived if in use)",
	})
}

// ToggleAccountStatus
// @Summary      Toggle Account Status
// @Description  Activates or deactivates an account. (Deactivating a default account unsets default).
// @Tags         Accounts
// @Accept       json
// @Produce      json
// @Param        tenantId  path      string  true  "Tenant UUID"
// @Param        accountId path      string  true  "Account UUID"
// @Param        request   body      map[string]bool true  "Status {is_active: true}"
// @Success      200       {object}  repo.Account
// @Failure      400       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts/{accountId}/status [patch]
func (h *AccountHandler) ToggleAccountStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	accountID := chi.URLParam(r, "accountId")

	var req map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "AccountHandler.ToggleStatus", "Invalid JSON format"))
		return
	}

	isActive, ok := req["is_active"]
	if !ok {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "AccountHandler.ToggleStatus", "Missing 'is_active' field"))
		return
	}

	acc, err := h.svc.ToggleAccountStatus(r.Context(), tenantID, accountID, isActive)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(acc)
}

// SetDefaultAccount
// @Summary      Set Default Account
// @Description  Marks an account as default and implicitly removes default from any other account in tenant.
// @Tags         Accounts
// @Produce      json
// @Param        tenantId  path      string  true  "Tenant UUID"
// @Param        accountId path      string  true  "Account UUID"
// @Param        request   body      map[string]bool false "Optional {is_default: true}"
// @Success      200       {object}  repo.Account
// @Failure      400       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts/{accountId}/default [patch]
func (h *AccountHandler) SetDefaultAccount(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	accountID := chi.URLParam(r, "accountId")

	acc, err := h.svc.SetDefaultAccount(r.Context(), tenantID, accountID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(acc)
}

// ListAccounts
// @Summary      List Accounts
// @Description  Returns a paginated list of accounts with optional filtering (search, status, type, currency).
// @Tags         Accounts
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        q            query     string  false  "Search by name or code"
// @Param        is_active    query     bool    false  "Filter by activity status"
// @Param        is_default   query     bool    false  "Filter by default status"
// @Param        account_type query     string  false  "Filter by type"
// @Param        currency     query     string  false  "Filter by currency"
// @Param        sort_by      query     string  false  "Sort column: created_at | name | code"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC"
// @Success      200          {object}  AccountListResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/accounts [get]
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	page := pagination.ParseRequest(r, "created_at", "updated_at", "name", "code")

	filters := repo.AccountFilters{}
	if q := r.URL.Query().Get("q"); q != "" {
		filters.Query = &q
	}
	if a := r.URL.Query().Get("is_active"); a != "" {
		active := a == "true"
		filters.IsActive = &active
	}
	if d := r.URL.Query().Get("is_default"); d != "" {
		def := d == "true"
		filters.IsDefault = &def
	}
	if at := r.URL.Query().Get("account_type"); at != "" {
		typ := domain.AccountType(at)
		filters.AccountType = &typ
	}
	if c := r.URL.Query().Get("currency"); c != "" {
		filters.Currency = &c
	}

	result, err := h.svc.ListAccounts(r.Context(), tenantID, filters, page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
