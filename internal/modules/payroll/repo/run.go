package payrollrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	"gorm.io/gorm"
)

// PayrollRun is the business entity returned by the repository layer.
type PayrollRun struct {
	ID          string                     `json:"id"`
	TenantID    string                     `json:"tenant_id"`
	LocationID  *string                    `json:"location_id"`
	Frequency   payrolldomain.RunFrequency `json:"frequency"`
	PeriodStart time.Time                  `json:"period_start"`
	PeriodEnd   time.Time                  `json:"period_end"`
	PayDate     time.Time                  `json:"pay_date"`
	Status      payrolldomain.RunStatus    `json:"status"`
	Total       float64                    `json:"total"`
	Currency    string                     `json:"currency"`
	Notes       *string                    `json:"notes"`
	CreatedBy   *string                    `json:"created_by"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
}

// UpdateRunFields holds optional fields that can be patched on a draft run.
type UpdateRunFields struct {
	LocationID  *string
	Frequency   *payrolldomain.RunFrequency
	PeriodStart *time.Time
	PeriodEnd   *time.Time
	PayDate     *time.Time
	Status      *payrolldomain.RunStatus
	Currency    *string
	Notes       *string
}

// RunFilters holds the allowed query filters
type RunFilters struct {
	Status      *payrolldomain.RunStatus
	Frequency   *payrolldomain.RunFrequency
	LocationID  *string
	PeriodStart *time.Time // period_start >=
	PeriodEnd   *time.Time // period_end <=
}

type PayrollRunRepo interface {
	Create(ctx context.Context, r *PayrollRun) error
	GetByID(ctx context.Context, tenantID, runID string) (*PayrollRun, error)
	Update(ctx context.Context, tenantID, runID string, fields UpdateRunFields) (*PayrollRun, error)
	Delete(ctx context.Context, tenantID, runID string) error
	List(ctx context.Context, tenantID string, filters RunFilters, page store.Page) (store.ResultList[PayrollRun], error)
}

type payrollRunRepo struct {
	db *gorm.DB
}

func NewPayrollRunRepo(db *gorm.DB) PayrollRunRepo {
	return &payrollRunRepo{db: db}
}

func mapRunToDomain(r *PayrollRun) *payrolldomain.PayrollRun {
	return &payrolldomain.PayrollRun{
		TenantID:    r.TenantID,
		LocationID:  r.LocationID,
		Frequency:   r.Frequency,
		PeriodStart: r.PeriodStart,
		PeriodEnd:   r.PeriodEnd,
		PayDate:     r.PayDate,
		Status:      r.Status,
		Total:       r.Total,
		Currency:    r.Currency,
		Notes:       r.Notes,
		CreatedBy:   r.CreatedBy,
	}
}

func mapRunToRepo(d payrolldomain.PayrollRun) PayrollRun {
	return PayrollRun{
		ID:          d.ID,
		TenantID:    d.TenantID,
		LocationID:  d.LocationID,
		Frequency:   d.Frequency,
		PeriodStart: d.PeriodStart,
		PeriodEnd:   d.PeriodEnd,
		PayDate:     d.PayDate,
		Status:      d.Status,
		Total:       d.Total,
		Currency:    d.Currency,
		Notes:       d.Notes,
		CreatedBy:   d.CreatedBy,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func (r *payrollRunRepo) Create(ctx context.Context, run *PayrollRun) error {
	model := mapRunToDomain(run)
	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return result.Error // Might be unique constraint violation
	}
	run.ID = model.ID
	run.CreatedAt = model.CreatedAt
	run.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *payrollRunRepo) GetByID(ctx context.Context, tenantID, runID string) (*PayrollRun, error) {
	var model payrolldomain.PayrollRun
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, runID)
	if result.Error != nil {
		return nil, result.Error
	}
	run := mapRunToRepo(model)
	return &run, nil
}

func (r *payrollRunRepo) Update(ctx context.Context, tenantID, runID string, fields UpdateRunFields) (*PayrollRun, error) {
	var model payrolldomain.PayrollRun
	updates := make(map[string]interface{})

	if fields.LocationID != nil {
		updates["location_id"] = *fields.LocationID
	}
	if fields.Frequency != nil {
		updates["frequency"] = *fields.Frequency
	}
	if fields.PeriodStart != nil {
		updates["period_start"] = *fields.PeriodStart
	}
	if fields.PeriodEnd != nil {
		updates["period_end"] = *fields.PeriodEnd
	}
	if fields.PayDate != nil {
		updates["pay_date"] = *fields.PayDate
	}
	if fields.Status != nil {
		updates["status"] = *fields.Status
	}
	if fields.Currency != nil {
		updates["currency"] = *fields.Currency
	}
	if fields.Notes != nil {
		updates["notes"] = *fields.Notes
	}

	result := r.db.WithContext(ctx).Model(&model).Where("tenant_id = ? AND id = ?", tenantID, runID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, runID)
	run := mapRunToRepo(model)
	return &run, nil
}

func (r *payrollRunRepo) Delete(ctx context.Context, tenantID, runID string) error {
	result := r.db.WithContext(ctx).Delete(&payrolldomain.PayrollRun{}, "tenant_id = ? AND id = ?", tenantID, runID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Allowed sorts matching domain
var runSortableColumns = map[string]bool{
	"created_at":   true,
	"updated_at":   true,
	"period_start": true,
	"period_end":   true,
	"pay_date":     true,
	"total":        true,
	"status":       true,
}

func (r *payrollRunRepo) List(ctx context.Context, tenantID string, filters RunFilters, page store.Page) (store.ResultList[PayrollRun], error) {
	orderBy := "period_start DESC"
	if len(page.Sorts) > 0 {
		s := page.Sorts[0]
		if runSortableColumns[s.Field] {
			dir := "DESC"
			if s.Direction == store.SortAsc {
				dir = "ASC"
			}
			orderBy = fmt.Sprintf("%s %s", s.Field, dir)
		}
	}

	query := r.db.WithContext(ctx).Model(&payrolldomain.PayrollRun{}).Where("tenant_id = ?", tenantID)

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.Frequency != nil {
		query = query.Where("frequency = ?", *filters.Frequency)
	}
	if filters.LocationID != nil {
		query = query.Where("location_id = ?", *filters.LocationID)
	}
	if filters.PeriodStart != nil {
		query = query.Where("period_start >= ?", *filters.PeriodStart)
	}
	if filters.PeriodEnd != nil {
		query = query.Where("period_end <= ?", *filters.PeriodEnd)
	}

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[PayrollRun]{}, countResult.Error
	}

	var models []payrolldomain.PayrollRun
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[PayrollRun]{}, result.Error
	}

	runs := make([]PayrollRun, len(models))
	for i, m := range models {
		runs[i] = mapRunToRepo(m)
	}

	return store.ResultList[PayrollRun]{
		Items: runs,
		Total: total,
		Page:  page,
	}, nil
}
