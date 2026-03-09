package payrollrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	"gorm.io/gorm"
)

// PayrollItem is the business entity returned by the repository layer.
type PayrollItem struct {
	ID            string                    `json:"id"`
	TenantID      string                    `json:"tenant_id"`
	PayrollRunID  string                    `json:"payroll_run_id"`
	StaffID       string                    `json:"staff_id"`
	LineType      payrolldomain.LineType    `json:"line_type"`
	Concept       payrolldomain.ItemConcept `json:"concept"`
	Amount        float64                   `json:"amount"`
	AppointmentID *string                   `json:"appointment_id"`
	ServiceID     *string                   `json:"service_id"`
	Notes         *string                   `json:"notes"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

type UpdateItemFields struct {
	LineType      *payrolldomain.LineType
	Concept       *payrolldomain.ItemConcept
	Amount        *float64
	AppointmentID *string
	ServiceID     *string
	Notes         *string
}

// ItemFilters holds the allowed query filters
type ItemFilters struct {
	StaffID  *string
	Concept  *payrolldomain.ItemConcept
	LineType *payrolldomain.LineType
}

type PayrollItemRepo interface {
	Create(ctx context.Context, item *PayrollItem) error
	GetByID(ctx context.Context, tenantID, runID, itemID string) (*PayrollItem, error)
	Update(ctx context.Context, tenantID, runID, itemID string, fields UpdateItemFields) (*PayrollItem, error)
	Delete(ctx context.Context, tenantID, runID, itemID string) error
	List(ctx context.Context, tenantID, runID string, filters ItemFilters, page store.Page) (store.ResultList[PayrollItem], error)
}

type payrollItemRepo struct {
	db *gorm.DB
}

func NewPayrollItemRepo(db *gorm.DB) PayrollItemRepo {
	return &payrollItemRepo{db: db}
}

func mapItemToDomain(i *PayrollItem) *payrolldomain.PayrollItem {
	return &payrolldomain.PayrollItem{
		TenantID:      i.TenantID,
		PayrollRunID:  i.PayrollRunID,
		StaffID:       i.StaffID,
		LineType:      i.LineType,
		Concept:       i.Concept,
		Amount:        i.Amount,
		AppointmentID: i.AppointmentID,
		ServiceID:     i.ServiceID,
		Notes:         i.Notes,
	}
}

func mapItemToRepo(d payrolldomain.PayrollItem) PayrollItem {
	return PayrollItem{
		ID:            d.ID,
		TenantID:      d.TenantID,
		PayrollRunID:  d.PayrollRunID,
		StaffID:       d.StaffID,
		LineType:      d.LineType,
		Concept:       d.Concept,
		Amount:        d.Amount,
		AppointmentID: d.AppointmentID,
		ServiceID:     d.ServiceID,
		Notes:         d.Notes,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

func (r *payrollItemRepo) Create(ctx context.Context, item *PayrollItem) error {
	model := mapItemToDomain(item)
	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return result.Error
	}
	item.ID = model.ID
	item.CreatedAt = model.CreatedAt
	item.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *payrollItemRepo) GetByID(ctx context.Context, tenantID, runID, itemID string) (*PayrollItem, error) {
	var model payrolldomain.PayrollItem
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND payroll_run_id = ? AND id = ?", tenantID, runID, itemID)
	if result.Error != nil {
		return nil, result.Error
	}
	item := mapItemToRepo(model)
	return &item, nil
}

func (r *payrollItemRepo) Update(ctx context.Context, tenantID, runID, itemID string, fields UpdateItemFields) (*PayrollItem, error) {
	var model payrolldomain.PayrollItem
	updates := make(map[string]interface{})

	if fields.LineType != nil {
		updates["line_type"] = *fields.LineType
	}
	if fields.Concept != nil {
		updates["concept"] = *fields.Concept
	}
	if fields.Amount != nil {
		updates["amount"] = *fields.Amount
	}
	if fields.AppointmentID != nil {
		updates["appointment_id"] = *fields.AppointmentID
	}
	if fields.ServiceID != nil {
		updates["service_id"] = *fields.ServiceID
	}
	if fields.Notes != nil {
		updates["notes"] = *fields.Notes
	}

	result := r.db.WithContext(ctx).Model(&model).Where("tenant_id = ? AND payroll_run_id = ? AND id = ?", tenantID, runID, itemID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND payroll_run_id = ? AND id = ?", tenantID, runID, itemID)
	item := mapItemToRepo(model)
	return &item, nil
}

func (r *payrollItemRepo) Delete(ctx context.Context, tenantID, runID, itemID string) error {
	result := r.db.WithContext(ctx).Delete(&payrolldomain.PayrollItem{}, "tenant_id = ? AND payroll_run_id = ? AND id = ?", tenantID, runID, itemID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Allowed sorts
var itemSortableColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"amount":     true,
	"line_type":  true,
}

func (r *payrollItemRepo) List(ctx context.Context, tenantID, runID string, filters ItemFilters, page store.Page) (store.ResultList[PayrollItem], error) {
	orderBy := "created_at DESC"
	if len(page.Sorts) > 0 {
		s := page.Sorts[0]
		if itemSortableColumns[s.Field] {
			dir := "DESC"
			if s.Direction == store.SortAsc {
				dir = "ASC"
			}
			orderBy = fmt.Sprintf("%s %s", s.Field, dir)
		}
	}

	query := r.db.WithContext(ctx).Model(&payrolldomain.PayrollItem{}).Where("tenant_id = ? AND payroll_run_id = ?", tenantID, runID)

	if filters.StaffID != nil {
		query = query.Where("staff_id = ?", *filters.StaffID)
	}
	if filters.Concept != nil {
		query = query.Where("concept = ?", *filters.Concept)
	}
	if filters.LineType != nil {
		query = query.Where("line_type = ?", *filters.LineType)
	}

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[PayrollItem]{}, countResult.Error
	}

	var models []payrolldomain.PayrollItem
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[PayrollItem]{}, result.Error
	}

	items := make([]PayrollItem, len(models))
	for i, m := range models {
		items[i] = mapItemToRepo(m)
	}

	return store.ResultList[PayrollItem]{
		Items: items,
		Total: total,
		Page:  page,
	}, nil
}
