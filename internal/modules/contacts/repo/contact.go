package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/contacts/domain"
	"gorm.io/gorm"
)

// Contact is the business entity returned by the repository layer.
type Contact struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Email          *string   `json:"email"`
	Phone          *string   `json:"phone"`
	CompanyName    *string   `json:"company_name"`
	ContactType    string    `json:"contact_type"`
	LifecycleStage string    `json:"lifecycle_stage"`
	Notes          *string   `json:"notes"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ContactSummary represents the aggregated statistics for a tenant's contacts.
type ContactSummary struct {
	Total     int64 `json:"total"`
	Active    int64 `json:"active"`
	Inactive  int64 `json:"inactive"`
	Leads     int64 `json:"leads"`
	Prospects int64 `json:"prospects"`
	Customers int64 `json:"customers"`
	Suppliers int64 `json:"suppliers"`
}

// UpdateFields holds the optional fields that can be patched on a Contact.
type UpdateFields struct {
	Name           *string
	Email          *string
	Phone          *string
	CompanyName    *string
	ContactType    *string
	LifecycleStage *string
	Notes          *string
}

// ContactRepo defines the persistence contract for contacts.
type ContactRepo interface {
	Create(ctx context.Context, c *Contact) error
	GetByID(ctx context.Context, tenantID, id string) (*Contact, error)
	Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Contact, error)
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID, contactTypeFilter string, page store.Page) (store.ResultList[Contact], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*Contact, error)
	UpdateLifecycleStage(ctx context.Context, tenantID, id, stage string) (*Contact, error)
	GetSummary(ctx context.Context, tenantID string) (*ContactSummary, error)
}

type contactRepo struct {
	db *gorm.DB
}

// NewContactRepo creates a repository backed by the given gorm.DB instance.
func NewContactRepo(db *gorm.DB) ContactRepo {
	return &contactRepo{db: db}
}

// mapToDomain converts the Repo Struct to the DB Struct for saving
func mapToDomain(c *Contact) *domain.Contact {
	return &domain.Contact{
		TenantID:       c.TenantID,
		Name:           c.Name,
		Email:          c.Email,
		Phone:          c.Phone,
		CompanyName:    c.CompanyName,
		ContactType:    c.ContactType,
		LifecycleStage: c.LifecycleStage,
		Notes:          c.Notes,
		IsActive:       c.IsActive,
	}
}

// mapToRepo converts the DB Struct to the returned Business Repo Struct
func mapToRepo(d domain.Contact) Contact {
	return Contact{
		ID:             d.ID,
		TenantID:       d.TenantID,
		Name:           d.Name,
		Email:          d.Email,
		Phone:          d.Phone,
		CompanyName:    d.CompanyName,
		ContactType:    d.ContactType,
		LifecycleStage: d.LifecycleStage,
		Notes:          d.Notes,
		IsActive:       d.IsActive,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *contactRepo) Create(ctx context.Context, c *Contact) error {
	model := mapToDomain(c)
	result := r.db.WithContext(ctx).Create(model)

	if result.Error != nil {
		return result.Error
	}

	c.ID = model.ID
	c.CreatedAt = model.CreatedAt
	c.UpdatedAt = model.UpdatedAt
	return nil
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func (r *contactRepo) GetByID(ctx context.Context, tenantID, id string) (*Contact, error) {
	var model domain.Contact
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)

	if result.Error != nil {
		return nil, result.Error
	}

	c := mapToRepo(model)
	return &c, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *contactRepo) Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Contact, error) {
	var model domain.Contact

	updates := make(map[string]interface{})
	if fields.Name != nil {
		updates["name"] = *fields.Name
	}
	if fields.Email != nil {
		updates["email"] = *fields.Email
	}
	if fields.Phone != nil {
		updates["phone"] = *fields.Phone
	}
	if fields.CompanyName != nil {
		updates["company_name"] = *fields.CompanyName
	}
	if fields.ContactType != nil {
		updates["contact_type"] = *fields.ContactType
	}
	if fields.LifecycleStage != nil {
		updates["lifecycle_stage"] = *fields.LifecycleStage
	}
	if fields.Notes != nil {
		updates["notes"] = *fields.Notes
	}

	result := r.db.WithContext(ctx).Model(&model).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)
	c := mapToRepo(model)
	return &c, nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func (r *contactRepo) Delete(ctx context.Context, tenantID, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Contact{}, "tenant_id = ? AND id = ?", tenantID, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ─── List ─────────────────────────────────────────────────────────────────────

var sortableColumns = map[string]bool{
	"created_at":      true,
	"updated_at":      true,
	"name":            true,
	"company_name":    true,
	"contact_type":    true,
	"lifecycle_stage": true,
	"is_active":       true,
}

func (r *contactRepo) List(ctx context.Context, tenantID, contactTypeFilter string, page store.Page) (store.ResultList[Contact], error) {
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

	query := r.db.WithContext(ctx).Model(&domain.Contact{}).Where("tenant_id = ?", tenantID)

	if contactTypeFilter != "" {
		query = query.Where("contact_type = ?", contactTypeFilter)
	}

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[Contact]{}, countResult.Error
	}

	var models []domain.Contact
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[Contact]{}, result.Error
	}

	contacts := make([]Contact, len(models))
	for i, m := range models {
		contacts[i] = mapToRepo(m)
	}

	return store.ResultList[Contact]{
		Items: contacts,
		Total: total,
		Page:  page,
	}, nil
}

// ─── ToggleStatus ─────────────────────────────────────────────────────────────

func (r *contactRepo) ToggleStatus(ctx context.Context, tenantID, id string) (*Contact, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE contacts
		SET is_active = NOT is_active, updated_at = now()
		WHERE tenant_id = ? AND id = ?
	`, tenantID, id)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return r.GetByID(ctx, tenantID, id)
}

// ─── UpdateLifecycleStage ─────────────────────────────────────────────────────

func (r *contactRepo) UpdateLifecycleStage(ctx context.Context, tenantID, id, stage string) (*Contact, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE contacts
		SET lifecycle_stage = ?, updated_at = now()
		WHERE tenant_id = ? AND id = ?
	`, stage, tenantID, id)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return r.GetByID(ctx, tenantID, id)
}

// ─── GetSummary ───────────────────────────────────────────────────────────────

func (r *contactRepo) GetSummary(ctx context.Context, tenantID string) (*ContactSummary, error) {
	var summary ContactSummary

	// We use a single query with conditional aggregation to compute all stats
	query := `
		SELECT 
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN is_active THEN 1 ELSE 0 END), 0) as active,
			COALESCE(SUM(CASE WHEN NOT is_active THEN 1 ELSE 0 END), 0) as inactive,
			COALESCE(SUM(CASE WHEN lifecycle_stage = 'lead' THEN 1 ELSE 0 END), 0) as leads,
			COALESCE(SUM(CASE WHEN lifecycle_stage = 'prospect' THEN 1 ELSE 0 END), 0) as prospects,
			COALESCE(SUM(CASE WHEN contact_type = 'customer' THEN 1 ELSE 0 END), 0) as customers,
			COALESCE(SUM(CASE WHEN contact_type = 'supplier' THEN 1 ELSE 0 END), 0) as suppliers
		FROM contacts
		WHERE tenant_id = ?
	`

	result := r.db.WithContext(ctx).Raw(query, tenantID).Scan(&summary)

	if result.Error != nil {
		return nil, result.Error
	}

	return &summary, nil
}
