package staffrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	staffdomain "github.com/Lovealone1/nex21-api/internal/modules/staff/domain"
	"gorm.io/gorm"
)

// Staff is the business entity returned by the repository layer.
type Staff struct {
	ID          string                `json:"id"`
	TenantID    string                `json:"tenant_id"`
	LocationID  *string               `json:"location_id"`
	ProfileID   *string               `json:"profile_id"`
	DisplayName string                `json:"display_name"`
	Email       *string               `json:"email"`
	Phone       *string               `json:"phone"`
	Role        staffdomain.StaffRole `json:"role"`
	IsActive    bool                  `json:"is_active"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// UpdateFields holds the optional fields that can be patched on a Staff member.
type UpdateFields struct {
	DisplayName *string
	Email       *string
	Phone       *string
	LocationID  *string
	ProfileID   *string
	Role        *staffdomain.StaffRole
	IsActive    *bool
}

// StaffRepo defines the persistence contract for staff.
type StaffRepo interface {
	Create(ctx context.Context, s *Staff) error
	GetByID(ctx context.Context, tenantID, id string) (*Staff, error)
	Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Staff, error)
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Staff], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*Staff, error)
}

type staffRepo struct {
	db *gorm.DB
}

// NewStaffRepo creates a repository backed by the given gorm.DB instance.
func NewStaffRepo(db *gorm.DB) StaffRepo {
	return &staffRepo{db: db}
}

// mapToDomain converts the Repo Struct to the DB Struct for saving
func mapToDomain(s *Staff) *staffdomain.Staff {
	return &staffdomain.Staff{
		TenantID:    s.TenantID,
		LocationID:  s.LocationID,
		ProfileID:   s.ProfileID,
		DisplayName: s.DisplayName,
		Email:       s.Email,
		Phone:       s.Phone,
		StaffRole:   s.Role,
		IsActive:    s.IsActive,
	}
}

// mapToRepo converts the DB Struct to the returned Business Repo Struct
func mapToRepo(d staffdomain.Staff) Staff {
	return Staff{
		ID:          d.ID,
		TenantID:    d.TenantID,
		LocationID:  d.LocationID,
		ProfileID:   d.ProfileID,
		DisplayName: d.DisplayName,
		Email:       d.Email,
		Phone:       d.Phone,
		Role:        d.StaffRole,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *staffRepo) Create(ctx context.Context, s *Staff) error {
	model := mapToDomain(s)
	result := r.db.WithContext(ctx).Create(model)

	if result.Error != nil {
		return result.Error
	}

	s.ID = model.ID
	s.CreatedAt = model.CreatedAt
	s.UpdatedAt = model.UpdatedAt
	return nil
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func (r *staffRepo) GetByID(ctx context.Context, tenantID, id string) (*Staff, error) {
	var model staffdomain.Staff
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)

	if result.Error != nil {
		return nil, result.Error
	}

	s := mapToRepo(model)
	return &s, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *staffRepo) Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Staff, error) {
	var model staffdomain.Staff

	updates := make(map[string]interface{})
	if fields.DisplayName != nil {
		updates["display_name"] = *fields.DisplayName
	}
	if fields.Email != nil {
		updates["email"] = *fields.Email
	}
	if fields.Phone != nil {
		updates["phone"] = *fields.Phone
	}
	if fields.LocationID != nil {
		updates["location_id"] = *fields.LocationID
	}
	if fields.ProfileID != nil {
		updates["profile_id"] = *fields.ProfileID
	}
	if fields.Role != nil {
		updates["staff_role"] = *fields.Role
	}
	if fields.IsActive != nil {
		updates["is_active"] = *fields.IsActive
	}

	result := r.db.WithContext(ctx).Model(&model).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)
	s := mapToRepo(model)
	return &s, nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func (r *staffRepo) Delete(ctx context.Context, tenantID, id string) error {
	result := r.db.WithContext(ctx).Delete(&staffdomain.Staff{}, "tenant_id = ? AND id = ?", tenantID, id)

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
	"created_at":   true,
	"updated_at":   true,
	"display_name": true,
	"staff_role":   true,
	"is_active":    true,
}

func (r *staffRepo) List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Staff], error) {
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

	query := r.db.WithContext(ctx).Model(&staffdomain.Staff{}).Where("tenant_id = ?", tenantID)

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[Staff]{}, countResult.Error
	}

	var models []staffdomain.Staff
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[Staff]{}, result.Error
	}

	staffList := make([]Staff, len(models))
	for i, m := range models {
		staffList[i] = mapToRepo(m)
	}

	return store.ResultList[Staff]{
		Items: staffList,
		Total: total,
		Page:  page,
	}, nil
}

// ─── ToggleStatus ─────────────────────────────────────────────────────────────

func (r *staffRepo) ToggleStatus(ctx context.Context, tenantID, id string) (*Staff, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE staff
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
