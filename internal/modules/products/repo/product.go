package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/products/domain"
	"gorm.io/gorm"
)

// Product is the business entity returned by the repository layer.
type Product struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	SKU         *string   `json:"sku"`
	Price       float64   `json:"price"`
	Currency    string    `json:"currency"`
	Quantity    int       `json:"quantity"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateFields holds the optional fields that can be patched on a Product.
// Note: Quantity is NOT here as it's managed via dedicated methods.
type UpdateFields struct {
	Name        *string
	Description *string
	SKU         *string
	Price       *float64
	Currency    *string
}

// ProductRepo defines the persistence contract for products.
type ProductRepo interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, tenantID, id string) (*Product, error)
	Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Product, error)
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Product], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*Product, error)

	// Stock management
	AdjustStock(ctx context.Context, tenantID, id string, delta int) error
	SetStock(ctx context.Context, tenantID, id string, quantity int) error
}

type productRepo struct {
	db *gorm.DB
}

// NewProductRepo creates a repository backed by the given gorm.DB instance.
func NewProductRepo(db *gorm.DB) ProductRepo {
	return &productRepo{db: db}
}

// mapToDomain converts the Repo Struct to the DB Struct for saving
func mapToDomain(p *Product) *domain.Product {
	return &domain.Product{
		TenantID:    p.TenantID,
		Name:        p.Name,
		Description: p.Description,
		SKU:         p.SKU,
		Price:       p.Price,
		Currency:    p.Currency,
		Quantity:    p.Quantity,
		IsActive:    p.IsActive,
	}
}

// mapToRepo converts the DB Struct to the returned Business Repo Struct
func mapToRepo(d domain.Product) Product {
	return Product{
		ID:          d.ID,
		TenantID:    d.TenantID,
		Name:        d.Name,
		Description: d.Description,
		SKU:         d.SKU,
		Price:       d.Price,
		Currency:    d.Currency,
		Quantity:    d.Quantity,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *productRepo) Create(ctx context.Context, p *Product) error {
	model := mapToDomain(p)
	result := r.db.WithContext(ctx).Create(model)

	if result.Error != nil {
		return result.Error
	}

	p.ID = model.ID
	p.CreatedAt = model.CreatedAt
	p.UpdatedAt = model.UpdatedAt
	return nil
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func (r *productRepo) GetByID(ctx context.Context, tenantID, id string) (*Product, error) {
	var model domain.Product
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)

	if result.Error != nil {
		return nil, result.Error
	}

	p := mapToRepo(model)
	return &p, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *productRepo) Update(ctx context.Context, tenantID, id string, fields UpdateFields) (*Product, error) {
	var model domain.Product

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
	if fields.Price != nil {
		updates["price"] = *fields.Price
	}
	if fields.Currency != nil {
		updates["currency"] = *fields.Currency
	}

	result := r.db.WithContext(ctx).Model(&model).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)
	p := mapToRepo(model)
	return &p, nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func (r *productRepo) Delete(ctx context.Context, tenantID, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Product{}, "tenant_id = ? AND id = ?", tenantID, id)

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
	"price":      true,
	"quantity":   true,
	"is_active":  true,
}

func (r *productRepo) List(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Product], error) {
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

	query := r.db.WithContext(ctx).Model(&domain.Product{}).Where("tenant_id = ?", tenantID)

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[Product]{}, countResult.Error
	}

	var models []domain.Product
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[Product]{}, result.Error
	}

	products := make([]Product, len(models))
	for i, m := range models {
		products[i] = mapToRepo(m)
	}

	return store.ResultList[Product]{
		Items: products,
		Total: total,
		Page:  page,
	}, nil
}

// ─── ToggleStatus ─────────────────────────────────────────────────────────────

func (r *productRepo) ToggleStatus(ctx context.Context, tenantID, id string) (*Product, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE products
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

// ─── Stock Management ──────────────────────────────────────────────────────────

func (r *productRepo) AdjustStock(ctx context.Context, tenantID, id string, delta int) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE products
		SET quantity = quantity + ?, updated_at = now()
		WHERE tenant_id = ? AND id = ?
	`, delta, tenantID, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *productRepo) SetStock(ctx context.Context, tenantID, id string, quantity int) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE products
		SET quantity = ?, updated_at = now()
		WHERE tenant_id = ? AND id = ?
	`, quantity, tenantID, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
