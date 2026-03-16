package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"gorm.io/gorm"
)

// Tenant is the business entity returned by the repository layer.
// All fields are JSON-tagged so handlers can encode them directly.
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Plan      string    `json:"plan"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateFields holds the optional fields that can be patched on a Tenant.
// Pointer types: only non-nil fields are written to the database.
type UpdateFields struct {
	Name     *string
	Slug     *string
	Plan     *string
	IsActive *bool
}

// TenantRepo defines the persistence contract for tenants.
type TenantRepo interface {
	// Create inserts a new tenant and returns the persisted entity.
	Create(ctx context.Context, t *Tenant) error
	// GetByID fetches a single tenant by its UUID.
	GetByID(ctx context.Context, id string) (*Tenant, error)
	// Update applies a partial patch to a tenant row.
	Update(ctx context.Context, id string, fields UpdateFields) (*Tenant, error)
	// Delete permanently removes a tenant row.
	Delete(ctx context.Context, id string) error
	// List returns a paginated, optionally sorted, list of tenants.
	List(ctx context.Context, page store.Page) (store.ResultList[Tenant], error)
}

type tenantRepo struct {
	db *gorm.DB
}

// NewTenantRepo creates a repository backed by the given gorm.DB instance.
func NewTenantRepo(db *gorm.DB) TenantRepo {
	return &tenantRepo{db: db}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *tenantRepo) Create(ctx context.Context, t *Tenant) error {
	result := r.db.WithContext(ctx).Create(t)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func (r *tenantRepo) GetByID(ctx context.Context, id string) (*Tenant, error) {
	var t Tenant
	result := r.db.WithContext(ctx).First(&t, "id = ?", id)

	if result.Error != nil {
		return nil, result.Error
	}
	return &t, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *tenantRepo) Update(ctx context.Context, id string, fields UpdateFields) (*Tenant, error) {
	var t Tenant

	// Map only fields that are provided
	updates := make(map[string]interface{})
	if fields.Name != nil {
		updates["name"] = *fields.Name
	}
	if fields.Slug != nil {
		updates["slug"] = *fields.Slug
	}
	if fields.Plan != nil {
		updates["plan"] = *fields.Plan
	}
	if fields.IsActive != nil {
		updates["is_active"] = *fields.IsActive
	}

	result := r.db.WithContext(ctx).Model(&t).Where("id = ?", id).Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// Fetch updated record back
	r.db.WithContext(ctx).First(&t, "id = ?", id)
	return &t, nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func (r *tenantRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Tenant{}, "id = ?", id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ─── List ─────────────────────────────────────────────────────────────────────

// sortableColumns is the allowlist for ORDER BY. Never interpolate user input
// without checking this map — prevents SQL injection.
var sortableColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"name":       true,
	"slug":       true,
	"plan":       true,
	"is_active":  true,
}

// List returns a paginated list of tenants.
// A single query with COUNT(*) OVER() avoids a second round-trip and prevents
// the pgx prepared-statement collision (42P05) that a separate COUNT query
// on the same connection would trigger.
func (r *tenantRepo) List(ctx context.Context, page store.Page) (store.ResultList[Tenant], error) {
	orderBy := "created_at DESC"
	if len(page.Sorts) > 0 {
		s := page.Sorts[0]
		if sortableColumns[s.Field] {
			dir := "DESC"
			if s.Direction == store.SortAsc {
				dir = "ASC"
			}
			orderBy = fmt.Sprintf("%s %s", s.Field, dir)
		}
	}

	var tenants []Tenant
	var total int64

	// Count total (avoids the COUNT+Find 42P05 issue by doing them on deeply separated queries)
	countResult := r.db.WithContext(ctx).Model(&Tenant{}).Count(&total)
	if countResult.Error != nil {
		return store.ResultList[Tenant]{}, countResult.Error
	}

	// Fetch data
	result := r.db.WithContext(ctx).Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&tenants)
	if result.Error != nil {
		return store.ResultList[Tenant]{}, result.Error
	}

	return store.ResultList[Tenant]{
		Items: tenants,
		Total: total,
		Page:  page,
	}, nil
}
