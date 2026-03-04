package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
	observability "github.com/Lovealone1/nex21-api/internal/platform/logger"
)

// Membership represents the mapping between a user and a tenant
type Membership struct {
	TenantID string `gorm:"column:tenant_id"`
	UserID   string `gorm:"column:user_id"`
	Status   string `gorm:"column:status"`
}

// TableName overrides the table name for GORM
func (Membership) TableName() string {
	return "public.memberships"
}

// TenantMiddleware constructs the middleware for handling X-Tenant-ID
func TenantMiddleware(database *db.Database) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get Tenant ID from header
			tenantID := r.Header.Get("X-Tenant-ID")

			if tenantID == "" {
				// We can optionally use URL Params if it's a specific route,
				// but forcing the Header is highly recommended for B2B APIs.
				tenantID = chi.URLParam(r, "tenantID")
				if tenantID == "" {
					http.Error(w, "X-Tenant-ID header is required", http.StatusUnauthorized)
					return
				}
			}

			// 2. Get User ID from Context (Injected by Auth Middleware)
			// Assuming your Auth middleware sets "user_id" in the context
			userID, ok := r.Context().Value("user_id").(string)
			if !ok || userID == "" {
				// Modo DEV: Si quieres probar esto en Postman sin Auth JWT todavía,
				// puedes comentar este bloque y setear un userID hardcodeado,
				// o pasar el userID por otro header (ej: X-User-ID) temporalmente.

				// DEV OVERRIDE (Remove in production if using real JWT):
				userID = r.Header.Get("X-Dev-User-ID")
				if userID == "" {
					http.Error(w, "Unauthorized: No user session found", http.StatusUnauthorized)
					return
				}
			}

			// 3. Verify Membership in Database
			var membership Membership
			// We DO NOT use db.TenantScope here because we are precisely in the step
			// of validating if we can grant that scope. We use GlobalScope or plain query.
			err := database.DB.WithContext(r.Context()).
				Where("tenant_id = ? AND user_id = ? AND status = 'active'", tenantID, userID).
				First(&membership).Error

			if err != nil {
				if err == gorm.ErrRecordNotFound {
					observability.Log.Warnf("Unauthorized tenant access attempt. User: %s, Tenant: %s", userID, tenantID)
					http.Error(w, "Forbidden: You don't have access to this tenant", http.StatusForbidden)
					return
				}

				observability.Log.Errorf("Error validating tenant membership: %v", err)
				http.Error(w, "Internal server error during tenant validation", http.StatusInternalServerError)
				return
			}

			// 4. Inject Verified Tenant ID into Context using our db package helper
			ctx := db.WithTenant(r.Context(), tenantID)

			// Continue with the new context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
