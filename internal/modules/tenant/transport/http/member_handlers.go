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

// MemberListResponse is the concrete paginated response for the members list endpoint.
type MemberListResponse struct {
	Items      []repo.Member `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int64         `json:"total_pages"`
}

// MemberHandler exposes the MemberService via REST endpoints.
type MemberHandler struct {
	svc service.MemberService
}

// NewMemberHandler creates the handler with its dependencies wired.
func NewMemberHandler(svc service.MemberService) *MemberHandler {
	return &MemberHandler{svc: svc}
}

// RegisterRoutes maps HTTP URLs for a tenant's members.
// This is typically mounted onto the router UNDER `/api/admin/v1/tenants/{id}/members`
func (h *MemberHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.AddMember)
	r.Get("/", h.ListMembers)
	r.Patch("/{userId}/role", h.UpdateRole)
	r.Patch("/{userId}/status", h.ToggleStatus)
	r.Delete("/{userId}", h.RemoveMember)
}

// AddMember
// @Summary      Add or Invite a user to a Tenant
// @Description  Adds an existing user (`user_id`) to a tenant with a given role.
// @Tags         Tenant Members
// @Accept       json
// @Produce      json
// @Param        id       path      string                   true  "Tenant UUID"
// @Param        request  body      service.AddMemberInput   true  "Member details (user_id and role)"
// @Success      201      {object}  repo.Member
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id}/members [post]
func (h *MemberHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")

	var req service.AddMemberInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "MemberHandler.Add", "Invalid JSON body format"))
		return
	}

	m, err := h.svc.AddMember(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(m)
}

// ListMembers
// @Summary      List all Members of a Tenant
// @Description  Returns a paginated list of all members for the specified tenant.
// @Tags         Tenant Members
// @Produce      json
// @Param        id        path      string  true   "Tenant UUID"
// @Param        page      query     int     false  "Page number (1-based, default: 1)"
// @Param        limit     query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by   query     string  false  "Sort column: created_at | updated_at | role | status | email | full_name"
// @Param        sort_dir  query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200       {object}  MemberListResponse
// @Failure      401       {object}  errors.HTTPErrorResponse
// @Failure      500       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id}/members [get]
func (h *MemberHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "role", "status", "email", "full_name",
	)

	result, err := h.svc.ListMembers(r.Context(), tenantID, page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdateRole
// @Summary      Change a member's role
// @Description  Updates a member's role inside the given tenant. Valid roles: owner, admin, staff, member.
// @Tags         Tenant Members
// @Accept       json
// @Produce      json
// @Param        id        path      string                        true  "Tenant UUID"
// @Param        userId    path      string                        true  "User UUID"
// @Param        request   body      service.UpdateMemberRoleInput true  "New Role"
// @Success      200       {object}  repo.Member
// @Failure      400       {object}  errors.HTTPErrorResponse
// @Failure      401       {object}  errors.HTTPErrorResponse
// @Failure      404       {object}  errors.HTTPErrorResponse
// @Failure      500       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id}/members/{userId}/role [patch]
func (h *MemberHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	var req service.UpdateMemberRoleInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "MemberHandler.UpdateRole", "Invalid JSON body format"))
		return
	}

	m, err := h.svc.UpdateRole(r.Context(), tenantID, userID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(m)
}

// ToggleStatus
// @Summary      Toggle a member's active status
// @Description  Automatically flips a member's status between "active" and "inactive" inside the tenant.
// @Tags         Tenant Members
// @Produce      json
// @Param        id        path      string  true  "Tenant UUID"
// @Param        userId    path      string  true  "User UUID"
// @Success      200       {object}  repo.Member
// @Failure      400       {object}  errors.HTTPErrorResponse
// @Failure      401       {object}  errors.HTTPErrorResponse
// @Failure      404       {object}  errors.HTTPErrorResponse
// @Failure      500       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id}/members/{userId}/status [patch]
func (h *MemberHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	m, err := h.svc.ToggleStatus(r.Context(), tenantID, userID)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(m)
}

// RemoveMember
// @Summary      Remove a member
// @Description  Permanently removes the user-tenant association. This action is irreversible.
// @Tags         Tenant Members
// @Produce      json
// @Param        id        path      string  true  "Tenant UUID"
// @Param        userId    path      string  true  "User UUID"
// @Success      200       {object}  map[string]string
// @Failure      400       {object}  errors.HTTPErrorResponse
// @Failure      401       {object}  errors.HTTPErrorResponse
// @Failure      404       {object}  errors.HTTPErrorResponse
// @Failure      500       {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{id}/members/{userId} [delete]
func (h *MemberHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	if err := h.svc.RemoveMember(r.Context(), tenantID, userID); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Member removed successfully",
	})
}
