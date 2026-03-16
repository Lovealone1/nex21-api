package payrollrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	"gorm.io/gorm"
)

// PayType enum representation for the API
type PayType string

const (
	PayTypeSalary               PayType = "salary"
	PayTypeHourly               PayType = "hourly"
	PayTypeSalaryPlusCommission PayType = "salary_plus_commission"
	PayTypeHourlyPlusCommission PayType = "hourly_plus_commission"
)

// PayFrequency enum representation for the API
type PayFrequency string

const (
	PayFrequencyMonthly    PayFrequency = "monthly"
	PayFrequencyBiweekly   PayFrequency = "biweekly"
	PayFrequencyWeekly     PayFrequency = "weekly"
	PayFrequencyDaily      PayFrequency = "daily"
	PayFrequencyPerService PayFrequency = "per_service"
)

// StaffPay is the business entity returned by the repository layer.
type StaffPay struct {
	ID             string       `json:"id"`
	TenantID       string       `json:"tenant_id"`
	StaffID        string       `json:"staff_id"`
	PayType        PayType      `json:"pay_type"`
	PayFrequency   PayFrequency `json:"pay_frequency"`
	Amount         float64      `json:"amount"` // base_salary or hourly_rate
	CommissionRate *float64     `json:"commission_rate"`
	StartDate      time.Time    `json:"start_date"`
	EndDate        *time.Time   `json:"end_date"`
	IsActive       bool         `json:"is_active"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// UpdateFields holds optional fields that can be patched.
type UpdateFields struct {
	PayType        *PayType
	PayFrequency   *PayFrequency
	Amount         *float64
	CommissionRate *float64
	StartDate      *time.Time
	EndDate        *time.Time
	IsActive       *bool
}

type StaffPayRepo interface {
	Create(ctx context.Context, s *StaffPay) error
	GetActive(ctx context.Context, tenantID, staffID string) (*StaffPay, error)
	GetByID(ctx context.Context, tenantID, staffID, payID string) (*StaffPay, error)
	Update(ctx context.Context, tenantID, staffID, payID string, fields UpdateFields) (*StaffPay, error)
	List(ctx context.Context, tenantID, staffID string, page store.Page) (store.ResultList[StaffPay], error)
	DeactivateActivePlan(ctx context.Context, tenantID, staffID string, effectiveTo time.Time) error
}

type staffPayRepo struct {
	db *gorm.DB
}

func NewStaffPayRepo(db *gorm.DB) StaffPayRepo {
	return &staffPayRepo{db: db}
}

func mapToDomain(s *StaffPay) *payrolldomain.StaffCompensation {
	domain := &payrolldomain.StaffCompensation{
		TenantID:      s.TenantID,
		StaffID:       s.StaffID,
		EffectiveFrom: s.StartDate,
		EffectiveTo:   s.EndDate,
		IsActive:      s.IsActive,
	}

	switch s.PayFrequency {
	case PayFrequencyPerService:
		domain.PayFrequency = "custom"
	default:
		domain.PayFrequency = string(s.PayFrequency)
	}

	switch s.PayType {
	case PayTypeSalary:
		domain.Scheme = "fixed"
		domain.BaseSalary = &s.Amount
	case PayTypeHourly:
		domain.Scheme = "hourly"
		domain.HourlyRate = &s.Amount
	case PayTypeSalaryPlusCommission:
		domain.Scheme = "mixed"
		domain.BaseSalary = &s.Amount
		domain.CommissionPct = s.CommissionRate
	case PayTypeHourlyPlusCommission:
		domain.Scheme = "mixed"
		domain.HourlyRate = &s.Amount
		domain.CommissionPct = s.CommissionRate
	}

	return domain
}

func mapToRepo(d payrolldomain.StaffCompensation) StaffPay {
	s := StaffPay{
		ID:             d.ID,
		TenantID:       d.TenantID,
		StaffID:        d.StaffID,
		CommissionRate: d.CommissionPct,
		StartDate:      d.EffectiveFrom,
		EndDate:        d.EffectiveTo,
		IsActive:       d.IsActive,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}

	switch d.PayFrequency {
	case "custom": // assuming custom maps back to per_service in our simple model
		s.PayFrequency = PayFrequencyPerService
	default:
		s.PayFrequency = PayFrequency(d.PayFrequency)
	}

	if d.Scheme == "fixed" {
		s.PayType = PayTypeSalary
		if d.BaseSalary != nil {
			s.Amount = *d.BaseSalary
		}
	} else if d.Scheme == "hourly" {
		s.PayType = PayTypeHourly
		if d.HourlyRate != nil {
			s.Amount = *d.HourlyRate
		}
	} else if d.Scheme == "mixed" {
		if d.BaseSalary != nil {
			s.PayType = PayTypeSalaryPlusCommission
			s.Amount = *d.BaseSalary
		} else if d.HourlyRate != nil {
			s.PayType = PayTypeHourlyPlusCommission
			s.Amount = *d.HourlyRate
		}
	}

	return s
}

func (r *staffPayRepo) Create(ctx context.Context, s *StaffPay) error {
	model := mapToDomain(s)
	// We might need to ensure the EffectiveFrom doesn't collide with existing plans
	// However, since we deactivate old plans first in the service layer, we are good.
	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return result.Error
	}
	s.ID = model.ID
	s.CreatedAt = model.CreatedAt
	s.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *staffPayRepo) GetActive(ctx context.Context, tenantID, staffID string) (*StaffPay, error) {
	var model payrolldomain.StaffCompensation
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND staff_id = ? AND is_active = ?", tenantID, staffID, true)
	if result.Error != nil {
		return nil, result.Error
	}
	s := mapToRepo(model)
	return &s, nil
}

func (r *staffPayRepo) GetByID(ctx context.Context, tenantID, staffID, payID string) (*StaffPay, error) {
	var model payrolldomain.StaffCompensation
	result := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND staff_id = ? AND id = ?", tenantID, staffID, payID)
	if result.Error != nil {
		return nil, result.Error
	}
	s := mapToRepo(model)
	return &s, nil
}

func (r *staffPayRepo) Update(ctx context.Context, tenantID, staffID, payID string, fields UpdateFields) (*StaffPay, error) {
	var model payrolldomain.StaffCompensation

	// Fetch current to merge safely
	err := r.db.WithContext(ctx).First(&model, "tenant_id = ? AND staff_id = ? AND id = ?", tenantID, staffID, payID).Error
	if err != nil {
		return nil, err
	}

	currentRepo := mapToRepo(model)

	if fields.PayType != nil {
		currentRepo.PayType = *fields.PayType
	}
	if fields.PayFrequency != nil {
		currentRepo.PayFrequency = *fields.PayFrequency
	}
	if fields.Amount != nil {
		currentRepo.Amount = *fields.Amount
	}
	if fields.CommissionRate != nil {
		currentRepo.CommissionRate = fields.CommissionRate
	}
	if fields.StartDate != nil {
		currentRepo.StartDate = *fields.StartDate
	}
	if fields.EndDate != nil {
		currentRepo.EndDate = fields.EndDate // can be nil
	}
	if fields.IsActive != nil {
		currentRepo.IsActive = *fields.IsActive
	}

	updatedModel := mapToDomain(&currentRepo)

	updates := map[string]interface{}{
		"scheme":         updatedModel.Scheme,
		"pay_frequency":  updatedModel.PayFrequency,
		"base_salary":    updatedModel.BaseSalary,
		"hourly_rate":    updatedModel.HourlyRate,
		"commission_pct": updatedModel.CommissionPct,
		"effective_from": updatedModel.EffectiveFrom,
		"effective_to":   updatedModel.EffectiveTo,
		"is_active":      updatedModel.IsActive,
	}

	result := r.db.WithContext(ctx).Model(&model).Where("tenant_id = ? AND staff_id = ? AND id = ?", tenantID, staffID, payID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}

	r.db.WithContext(ctx).First(&model, "tenant_id = ? AND staff_id = ? AND id = ?", tenantID, staffID, payID)
	s := mapToRepo(model)
	return &s, nil
}

func (r *staffPayRepo) List(ctx context.Context, tenantID, staffID string, page store.Page) (store.ResultList[StaffPay], error) {
	orderBy := "created_at DESC"
	if len(page.Sorts) > 0 {
		s := page.Sorts[0]
		dir := "DESC"
		if s.Direction == store.SortAsc {
			dir = "ASC"
		}
		orderBy = fmt.Sprintf("%s %s", s.Field, dir) // Caution with SQL injection, using standard fields
	}

	query := r.db.WithContext(ctx).Model(&payrolldomain.StaffCompensation{}).Where("tenant_id = ? AND staff_id = ?", tenantID, staffID)

	var total int64
	countResult := query.Count(&total)
	if countResult.Error != nil {
		return store.ResultList[StaffPay]{}, countResult.Error
	}

	var models []payrolldomain.StaffCompensation
	result := query.Order(orderBy).Offset(page.Offset).Limit(page.Limit).Find(&models)
	if result.Error != nil {
		return store.ResultList[StaffPay]{}, result.Error
	}

	list := make([]StaffPay, len(models))
	for i, m := range models {
		list[i] = mapToRepo(m)
	}

	return store.ResultList[StaffPay]{
		Items: list,
		Total: total,
		Page:  page,
	}, nil
}

// DeactivateActivePlan deactivates an active plan and sets its effective_to to prevent overlap
func (r *staffPayRepo) DeactivateActivePlan(ctx context.Context, tenantID, staffID string, effectiveTo time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE staff_compensation
		SET is_active = false, effective_to = ?
		WHERE tenant_id = ? AND staff_id = ? AND is_active = true
	`, effectiveTo.Format("2006-01-02"), tenantID, staffID).Error
}
