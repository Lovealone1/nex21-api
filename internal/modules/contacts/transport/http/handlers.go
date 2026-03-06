package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/contacts/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/contacts/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// ContactListResponse represents a paginated list of contacts
type ContactListResponse struct {
	Items      []repo.Contact `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int64          `json:"total_pages"`
}

type ContactHandler struct {
	svc service.ContactService
}

func NewContactHandler(svc service.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
}

// RegisterRoutes sets up the REST endpoints for contacts.
// This is typically mounted onto the router UNDER `/api/admin/v1/tenants/{tenantId}/contacts`
func (h *ContactHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateContact)
	r.Get("/", h.ListContacts)
	r.Get("/summary", h.GetSummary) // MUST be before {id}
	r.Get("/{id}", h.GetContactByID)
	r.Patch("/{id}", h.UpdateContact)
	r.Delete("/{id}", h.DeleteContact)
	r.Patch("/{id}/status", h.ToggleStatus)
	r.Patch("/{id}/lifecycle", h.UpdateLifecycleStage)
}

// CreateContact
// @Summary      Create a new Contact
// @Description  Creates a new customer or supplier in the specified tenant workspace.
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                     true  "Tenant UUID"
// @Param        request  body      service.CreateContactInput true  "Contact data"
// @Success      201      {object}  repo.Contact
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts [post]
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req service.CreateContactInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ContactHandler.Create", "Invalid JSON format"))
		return
	}

	c, err := h.svc.CreateContact(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}

// GetContactByID
// @Summary      Get a Contact by ID
// @Description  Returns the contact details if found inside the tenant.
// @Tags         Contacts
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Contact UUID"
// @Success      200      {object}  repo.Contact
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts/{id} [get]
func (h *ContactHandler) GetContactByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	c, err := h.svc.GetContactByID(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c)
}

// UpdateContact
// @Summary      Update a Contact
// @Description  Applies a partial update to a contact. Only non-null fields will be modified.
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                     true  "Tenant UUID"
// @Param        id       path      string                     true  "Contact UUID"
// @Param        request  body      service.UpdateContactInput true  "Fields to update"
// @Success      200      {object}  repo.Contact
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts/{id} [patch]
func (h *ContactHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req service.UpdateContactInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ContactHandler.Update", "Invalid JSON format"))
		return
	}

	c, err := h.svc.UpdateContact(r.Context(), tenantID, id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c)
}

// DeleteContact
// @Summary      Delete a Contact
// @Description  Permanently deletes a contact from the tenant.
// @Tags         Contacts
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Contact UUID"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts/{id} [delete]
func (h *ContactHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	if err := h.svc.DeleteContact(r.Context(), tenantID, id); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Contact deleted successfully",
	})
}

// ListContacts
// @Summary      List all Contacts for a Tenant
// @Description  Returns a paginated list of contacts, optionally filtered by `contact_type` (customer, supplier, both).
// @Tags         Contacts
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        contact_type query     string  false  "Filter by type: customer, supplier, both"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by      query     string  false  "Sort column: created_at | name | company_name | contact_type | is_active | lifecycle_stage"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200          {object}  ContactListResponse
// @Failure      400          {object}  errors.HTTPErrorResponse
// @Failure      401          {object}  errors.HTTPErrorResponse
// @Failure      500          {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts [get]
func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	contactType := r.URL.Query().Get("contact_type")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "name", "company_name", "contact_type", "is_active", "lifecycle_stage",
	)

	result, err := h.svc.ListContacts(r.Context(), tenantID, contactType, page)
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
// @Summary      Toggle Contact Status
// @Description  Automatically flips a contact's is_active status boolean.
// @Tags         Contacts
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Contact UUID"
// @Success      200      {object}  repo.Contact
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts/{id}/status [patch]
func (h *ContactHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	c, err := h.svc.ToggleStatus(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c)
}

// UpdateLifecycleStage
// @Summary      Update Contact Lifecycle Stage
// @Description  Changes the lifecycle stage of a contact (e.g., lead to prospect).
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Contact UUID"
// @Param        request  body      map[string]string true "New stage like {\"lifecycle_stage\": \"prospect\"}"
// @Success      200      {object}  repo.Contact
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts/{id}/lifecycle [patch]
func (h *ContactHandler) UpdateLifecycleStage(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req struct {
		LifecycleStage string `json:"lifecycle_stage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ContactHandler.UpdateLifecycleStage", "Invalid JSON format"))
		return
	}

	c, err := h.svc.UpdateLifecycleStage(r.Context(), tenantID, id, req.LifecycleStage)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c)
}

// GetSummary
// @Summary      Get Contacts Summary
// @Description  Returns aggregated statistics for tenant's contacts.
// @Tags         Contacts
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Success      200      {object}  repo.ContactSummary
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/contacts/summary [get]
func (h *ContactHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	summary, err := h.svc.GetContactSummary(r.Context(), tenantID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(summary)
}
