package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/locations/domain"
	"gorm.io/gorm"
)

// Location is the business entity returned by the repository layer.
type Location struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Code      *string   `json:"code"`
	Phone     *string   `json:"phone"`
	Email     *string   `json:"email"`
	Address   *string   `json:"address"`
	City      *string   `json:"city"`
	Country   *string   `json:"country"`
	IsActive  bool      `json:"is_active"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateFields holds the optional fields that can be patched on a Location.
type UpdateFields struct {
	Name      *string
	Code      *string
	Phone     *string
	Email     *string
	Address   *string
	City      *string
	Country   *string
	IsActive  *bool
	IsDefault *bool
}

// LocationRepo defines the persistence contract for locations.
type LocationRepo interface {
	Create(ctx context.Context, l *Location) error
	GetByID(ctx context.Context, tenantID, id string) (*Location, error)
	Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Location, error)
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Location], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*Location, error)
}

type locationRepo struct {
	db *gorm.DB
}

// NewLocationRepo creates a repository backed by the given gorm.DB instance.
func NewLocationRepo(db *gorm.DB) LocationRepo {
	return &locationRepo{db: db}
}

// mapToDomain converts the Repo Struct to the DB Struct for saving
func mapToDomain(l *Location) *domain.Location {
	return &domain.Location{
		TenantID:  l.TenantID,
		Name:      l.Name,
		Code:      l.Code,
		Phone:     l.Phone,
		Email:     l.Email,
		Address:   l.Address,
		City:      l.City,
		Country:   l.Country,
		IsActive:  l.IsActive,
		IsDefault: l.IsDefault,
	}
}

// mapToRepo converts the DB Struct to the returned Business Repo Struct
func mapToRepo(d domain.Location) Location {
	return Location{
		ID:        d.ID,
		TenantID:  d.TenantID,
		Name:      d.Name,
		Code:      d.Code,
		Phone:     d.Phone,
		Email:     d.Email,
		Address:   d.Address,
		City:      d.City,
		Country:   d.Country,
		IsActive:  d.IsActive,
		IsDefault: d.IsDefault,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *locationRepo) Create(ctx context.Context, l *Location) error {
	model := mapToDomain(l)
	result := r.db.WithContext(ctx).Create(model)

	if result.Error != nil {
		return result.Error
	}

	l.ID = model.ID
	l.CreatedAt = model.CreatedAt
	l.UpdatedAt = model.UpdatedAt
	return nil
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func (r *locationRepo) GetByID(ctx context.Context, tenantID, id string) (*Location, error) {
	var model domain.Location
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)

	if result.Error != nil {
		return nil, result.Error
	}

	l := mapToRepo(model)
	return &l, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *locationRepo) Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Location, error) {
	var model domain.Location

	updates := make(map[string]interface{})
	if fields.Name != nil {
		updates["name"] = *fields.Name
	}
	if fields.Code != nil {
		updates["code"] = *fields.Code
	}
	if fields.Phone != nil {
		updates["phone"] = *fields.Phone
	}
	if fields.Email != nil {
		updates["email"] = *fields.Email
	}
	if fields.Address != nil {
		updates["address"] = *fields.Address
	}
	if fields.City != nil {
		updates["city"] = *fields.City
	}
	if fields.Country != nil {
		updates["country"] = *fields.Country
	}
	if fields.IsActive != nil {
		updates["is_active"] = *fields.IsActive
	}
	if fields.IsDefault != nil {
		updates["is_default"] = *fields.IsDefault
	}

	result := r.db.WithContext(ctx).Model(&model).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)
	l := mapToRepo(model)
	return &l, nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func (r *locationRepo) Delete(ctx context.Context, tenantID, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Location{}, "tenant_id = ? AND id = ?", tenantID, id)

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
	"created_at": true,
	"updated_at": true,
	"name":       true,
	"code":       true,
	"is_active":  true,
}

func (r *locationRepo) List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Location], error) {
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

	query := r.db.WithContext(ctx).Model(&domain.Location{}).Where("tenant_id = ?", tenantID)

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[Location]{}, countResult.Error
	}

	var models []domain.Location
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[Location]{}, result.Error
	}

	locations := make([]Location, len(models))
	for i, m := range models {
		locations[i] = mapToRepo(m)
	}

	return store.ResultList[Location]{
		Items: locations,
		Total: total,
		Page:  page,
	}, nil
}

// ─── ToggleStatus ─────────────────────────────────────────────────────────────

func (r *locationRepo) ToggleStatus(ctx context.Context, tenantID, id string) (*Location, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE locations
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
