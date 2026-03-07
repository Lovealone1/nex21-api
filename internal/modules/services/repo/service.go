package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/services/domain"
	"gorm.io/gorm"
)

// Service is the business entity returned by the repository layer.
type Service struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	SKU             *string   `json:"sku"`
	DurationMinutes int       `json:"duration_minutes"`
	BufferMinutes   int       `json:"buffer_minutes"`
	Price           float64   `json:"price"`
	Currency        string    `json:"currency"`
	Category        *string   `json:"category"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UpdateFields holds the optional fields that can be patched on a Service.
type UpdateFields struct {
	Name            *string
	Description     *string
	SKU             *string
	DurationMinutes *int
	BufferMinutes   *int
	Price           *float64
	Currency        *string
	Category        *string
}

// ServiceRepo defines the persistence contract for services.
type ServiceRepo interface {
	Create(ctx context.Context, s *Service) error
	GetByID(ctx context.Context, tenantID, id string) (*Service, error)
	Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Service, error)
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Service], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*Service, error)
}

type serviceRepo struct {
	db *gorm.DB
}

// NewServiceRepo creates a repository backed by the given gorm.DB instance.
func NewServiceRepo(db *gorm.DB) ServiceRepo {
	return &serviceRepo{db: db}
}

// mapToDomain converts the Repo Struct to the DB Struct for saving
func mapToDomain(s *Service) *domain.Service {
	return &domain.Service{
		TenantID:        s.TenantID,
		Name:            s.Name,
		Description:     s.Description,
		SKU:             s.SKU,
		DurationMinutes: s.DurationMinutes,
		BufferMinutes:   s.BufferMinutes,
		Price:           s.Price,
		Currency:        s.Currency,
		Category:        s.Category,
		IsActive:        s.IsActive,
	}
}

// mapToRepo converts the DB Struct to the returned Business Repo Struct
func mapToRepo(d domain.Service) Service {
	return Service{
		ID:              d.ID,
		TenantID:        d.TenantID,
		Name:            d.Name,
		Description:     d.Description,
		SKU:             d.SKU,
		DurationMinutes: d.DurationMinutes,
		BufferMinutes:   d.BufferMinutes,
		Price:           d.Price,
		Currency:        d.Currency,
		Category:        d.Category,
		IsActive:        d.IsActive,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *serviceRepo) Create(ctx context.Context, s *Service) error {
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

func (r *serviceRepo) GetByID(ctx context.Context, tenantID, id string) (*Service, error) {
	var model domain.Service
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)

	if result.Error != nil {
		return nil, result.Error
	}

	s := mapToRepo(model)
	return &s, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *serviceRepo) Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Service, error) {
	var model domain.Service

	updates := make(map[string]interface{})
	if fields.Name != nil {
		updates["name"] = *fields.Name
	}
	if fields.Description != nil {
		updates["description"] = *fields.Description
	}
	if fields.SKU != nil {
		updates["sku"] = *fields.SKU
	}
	if fields.DurationMinutes != nil {
		updates["duration_minutes"] = *fields.DurationMinutes
	}
	if fields.BufferMinutes != nil {
		updates["buffer_minutes"] = *fields.BufferMinutes
	}
	if fields.Price != nil {
		updates["price"] = *fields.Price
	}
	if fields.Currency != nil {
		updates["currency"] = *fields.Currency
	}
	if fields.Category != nil {
		updates["category"] = *fields.Category
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

func (r *serviceRepo) Delete(ctx context.Context, tenantID, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Service{}, "tenant_id = ? AND id = ?", tenantID, id)

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
	"created_at":       true,
	"updated_at":       true,
	"name":             true,
	"price":            true,
	"category":         true,
	"is_active":        true,
	"duration_minutes": true,
	"sku":              true,
}

func (r *serviceRepo) List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Service], error) {
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

	query := r.db.WithContext(ctx).Model(&domain.Service{}).Where("tenant_id = ?", tenantID)

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[Service]{}, countResult.Error
	}

	var models []domain.Service
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[Service]{}, result.Error
	}

	services := make([]Service, len(models))
	for i, m := range models {
		services[i] = mapToRepo(m)
	}

	return store.ResultList[Service]{
		Items: services,
		Total: total,
		Page:  page,
	}, nil
}

// ─── ToggleStatus ─────────────────────────────────────────────────────────────

func (r *serviceRepo) ToggleStatus(ctx context.Context, tenantID, id string) (*Service, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE services
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
