package db

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const TenantKey contextKey = "tenant_id"

var ErrMissingTenant = errors.New("security block: missing tenant_id in context")

// WithTenant is a helper to inject the tenant_id into a context.
// Typically used in your Tenant Middleware.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantKey, tenantID)
}

// ExtractTenant safely extracts the tenant_id from the context.
func ExtractTenant(ctx context.Context) string {
	if val, ok := ctx.Value(TenantKey).(string); ok {
		return val
	}
	return ""
}

// TenantScope is the ineludible GORM scope that enforces multi-tenancy.
// Apply this to all DB operations on tenant-owned tables.
func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID := ExtractTenant(ctx)

		if tenantID == "" {
			// INELUDIBLE CHECK: If someone forgot to pass a context with a tenant,
			// the query immediately fails and prevents accidental cross-tenant leaks.
			_ = db.AddError(ErrMissingTenant)
			// Return a tautology that is false so no rows are returned even if the error was ignored
			return db.Where("1 = 0")
		}

		// Success: Inject the WHERE tenant_id = '...' automatically
		return db.Where("tenant_id = ?", tenantID)
	}
}

// GlobalScope explicitly denotes a query over a global table (e.g. users, countries).
// Optionally acts as a semantic marker that this query IS NOT tenant-scoped.
func GlobalScope() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db
	}
}
