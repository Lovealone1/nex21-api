package middleware

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
	"github.com/Lovealone1/nex21-api/internal/platform/observability"
)

// StandardError payload
type StandardError struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	TraceID string `json:"trace_id"`
}

var tenantRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

func respondError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(StandardError{
		Error:   msg,
		Code:    status,
		TraceID: middleware.GetReqID(r.Context()),
	})
}

// ExtractTenantID resolves the tenant ID from Subdomain, then Headers
func ExtractTenantID(r *http.Request) string {
	// 1. Try Subdomain (Host check)
	host := r.Host
	if strings.Contains(host, "nex21.com") { // Adjust for env
		parts := strings.Split(host, ".")
		if len(parts) >= 3 && parts[0] != "api" {
			sub := parts[0]
			if sub == "www" && len(parts) >= 4 {
				sub = parts[1]
			}
			if sub != "www" && sub != "api" {
				return strings.ToLower(strings.TrimSpace(sub))
			}
		}
	}

	// 2. Try Header
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID != "" {
		return strings.ToLower(strings.TrimSpace(tenantID))
	}

	return ""
}

func TenantMiddleware(database *db.Database) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract and Validate Format
			tenantID := ExtractTenantID(r)
			if tenantID == "" || !tenantRegex.MatchString(tenantID) {
				respondError(w, r, http.StatusBadRequest, "Invalid or missing Tenant ID")
				return
			}

			// Extract UserID from previous Auth Middleware
			userID, ok := r.Context().Value("user_id").(string) // Ajustar según tu Auth actual
			if !ok || userID == "" {
				// Fallback to DEV override (Remove in Prod)
				userID = r.Header.Get("X-Dev-User-ID")
				if userID == "" {
					respondError(w, r, http.StatusUnauthorized, "Missing user session")
					return
				}
			}

			// Validate Membership (Using Global Scope query as this is the gateway)
			// Assuming you have a struct db.Membership or similar
			var role struct{ Role string }
			err := database.DB.WithContext(r.Context()).
				Table("public.memberships").
				Select("role").
				Where("tenant_id = ? AND user_id = ? AND status = 'active'", tenantID, userID).
				Scan(&role).Error

			if err != nil || role.Role == "" {
				if err == gorm.ErrRecordNotFound || role.Role == "" {
					observability.Log.Warnf("Unauthorized tenant access: User %s, Tenant %s", userID, tenantID)
					respondError(w, r, http.StatusForbidden, "Forbidden: No access to this tenant")
					return
				}
				observability.Log.Errorf("DB error validating membership: %v", err)
				respondError(w, r, http.StatusInternalServerError, "Internal Server Error")
				return
			}

			// Build the Actor and Inject to Context
			actor := db.Actor{
				UserID:   userID,
				TenantID: tenantID,
				Role:     role.Role,
			}
			ctx := db.WithActor(r.Context(), actor)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
