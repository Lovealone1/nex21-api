package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/finance/domain"
	"gorm.io/gorm"
)

type Account struct {
	ID          string             `json:"id"`
	TenantID    string             `json:"tenant_id"`
	Name        string             `json:"name"`
	Code        *string            `json:"code"`
	AccountType domain.AccountType `json:"account_type"`
	Currency    string             `json:"currency"`
	IsActive    bool               `json:"is_active"`
	IsDefault   bool               `json:"is_default"`
	Provider    *string            `json:"provider"`
	Notes       *string            `json:"notes"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type UpdateAccountFields struct {
	Name        *string
	Code        *string
	AccountType *domain.AccountType
	Currency    *string
	IsActive    *bool
	Provider    *string
	Notes       *string
}

type AccountFilters struct {
	IsActive    *bool
	IsDefault   *bool
	AccountType *domain.AccountType
	Currency    *string
	Query       *string // searches name or code
}

type AccountRepo interface {
	Create(ctx context.Context, acc *Account) error
	GetByID(ctx context.Context, tenantID, id string) (*Account, error)
	GetDefault(ctx context.Context, tenantID string) (*Account, error)
	Update(ctx context.Context, tenantID, id string, fields UpdateAccountFields) (*Account, error)
	Delete(ctx context.Context, tenantID, id string) error // Returns specific error on FK violation
	List(ctx context.Context, tenantID string, filters AccountFilters, page store.Page) (store.ResultList[Account], error)
	SetDefaultTx(ctx context.Context, tenantID, accountID string) error
	ToggleStatus(ctx context.Context, tenantID, accountID string, isActive bool) (*Account, error)
}

type accountRepo struct {
	db *gorm.DB
}

func NewAccountRepo(db *gorm.DB) AccountRepo {
	return &accountRepo{db: db}
}

func mapToDomain(a *Account) *domain.Account {
	return &domain.Account{
		TenantID:    a.TenantID,
		Name:        a.Name,
		Code:        a.Code,
		AccountType: a.AccountType,
		Currency:    a.Currency,
		IsActive:    a.IsActive,
		IsDefault:   a.IsDefault,
		Provider:    a.Provider,
		Notes:       a.Notes,
	}
}

func mapToRepo(d domain.Account) Account {
	return Account{
		ID:          d.ID,
		TenantID:    d.TenantID,
		Name:        d.Name,
		Code:        d.Code,
		AccountType: d.AccountType,
		Currency:    d.Currency,
		IsActive:    d.IsActive,
		IsDefault:   d.IsDefault,
		Provider:    d.Provider,
		Notes:       d.Notes,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func (r *accountRepo) Create(ctx context.Context, acc *Account) error {
	model := mapToDomain(acc)
	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return result.Error // Might be unique constraint violation (codigo)
	}
	acc.ID = model.ID
	acc.CreatedAt = model.CreatedAt
	acc.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *accountRepo) GetByID(ctx context.Context, tenantID, id string) (*Account, error) {
	var model domain.Account
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return nil, result.Error
	}
	acc := mapToRepo(model)
	return &acc, nil
}

func (r *accountRepo) GetDefault(ctx context.Context, tenantID string) (*Account, error) {
	var model domain.Account
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND is_default = ?", tenantID, true)
	if result.Error != nil {
		return nil, result.Error
	}
	acc := mapToRepo(model)
	return &acc, nil
}

func (r *accountRepo) Update(ctx context.Context, tenantID, id string, fields UpdateAccountFields) (*Account, error) {
	var model domain.Account
	updates := make(map[string]interface{})

	if fields.Name != nil {
		updates["name"] = *fields.Name
	}
	if fields.Code != nil {
		updates["code"] = *fields.Code
	}
	if fields.AccountType != nil {
		updates["account_type"] = *fields.AccountType
	}
	if fields.Currency != nil {
		updates["currency"] = *fields.Currency
	}
	if fields.IsActive != nil {
		updates["is_active"] = *fields.IsActive
	}
	if fields.Provider != nil {
		updates["provider"] = *fields.Provider
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
	acc := mapToRepo(model)
	return &acc, nil
}

func (r *accountRepo) Delete(ctx context.Context, tenantID, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Account{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error // Handled by service layer (checks for 23503 FK error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SetDefaultTx uses a transaction to atomically unset any existing default and set the new one
func (r *accountRepo) SetDefaultTx(ctx context.Context, tenantID, accountID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Unset existing default in this tenant
		if err := tx.Model(&domain.Account{}).
			Where("tenant_id = ? AND is_default = ?", tenantID, true).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// 2. Set the new default
		result := tx.Model(&domain.Account{}).
			Where("tenant_id = ? AND id = ?", tenantID, accountID).
			Update("is_default", true)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (r *accountRepo) ToggleStatus(ctx context.Context, tenantID, accountID string, isActive bool) (*Account, error) {
	var model domain.Account
	// If deactivating, also unset is_default natively
	updates := map[string]interface{}{"is_active": isActive}
	if !isActive {
		updates["is_default"] = false
	}

	result := r.db.WithContext(ctx).Model(&model).
		Where("tenant_id = ? AND id = ?", tenantID, accountID).
		Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, accountID)
	acc := mapToRepo(model)
	return &acc, nil
}

var accSortableColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"name":       true,
	"code":       true,
}

func (r *accountRepo) List(ctx context.Context, tenantID string, filters AccountFilters, page store.Page) (store.ResultList[Account], error) {
	orderBy := "created_at DESC"
	if len(page.Sorts) > 0 {
		s := page.Sorts[0]
		if accSortableColumns[s.Field] {
			dir := "DESC"
			if s.Direction == store.SortAsc {
				dir = "ASC"
			}
			orderBy = fmt.Sprintf("%s %s", s.Field, dir)
		}
	}

	query := r.db.WithContext(ctx).Model(&domain.Account{}).Where("tenant_id = ?", tenantID)

	if filters.IsActive != nil {
		query = query.Where("is_active = ?", *filters.IsActive)
	}
	if filters.IsDefault != nil {
		query = query.Where("is_default = ?", *filters.IsDefault)
	}
	if filters.AccountType != nil {
		query = query.Where("account_type = ?", *filters.AccountType)
	}
	if filters.Currency != nil {
		query = query.Where("currency = ?", *filters.Currency)
	}
	if filters.Query != nil && *filters.Query != "" {
		searchTerm := "%" + strings.ToLower(*filters.Query) + "%"
		query = query.Where("(lower(name) LIKE ? OR lower(code) LIKE ?)", searchTerm, searchTerm)
	}

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[Account]{}, countResult.Error
	}

	var models []domain.Account
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[Account]{}, result.Error
	}

	items := make([]Account, len(models))
	for i, m := range models {
		items[i] = mapToRepo(m)
	}

	return store.ResultList[Account]{
		Items: items,
		Total: total,
		Page:  page,
	}, nil
}
