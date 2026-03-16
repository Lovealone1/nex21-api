package payrollservice

import (
	"context"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type CreateStaffPayInput struct {
	PayType        payrollrepo.PayType      `json:"pay_type"`
	PayFrequency   payrollrepo.PayFrequency `json:"pay_frequency"`
	Amount         float64                  `json:"amount"`
	CommissionRate *float64                 `json:"commission_rate,omitempty"`
	StartDate      string                   `json:"start_date"` // YYYY-MM-DD
	EndDate        *string                  `json:"end_date,omitempty"`
}

type UpdateStaffPayInput struct {
	PayType        *payrollrepo.PayType      `json:"pay_type,omitempty"`
	PayFrequency   *payrollrepo.PayFrequency `json:"pay_frequency,omitempty"`
	Amount         *float64                  `json:"amount,omitempty"`
	CommissionRate *float64                  `json:"commission_rate,omitempty"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

type StaffPayService interface {
	GetActiveStaffPay(ctx context.Context, tenantID, staffID string) (*payrollrepo.StaffPay, error)
	CreateStaffPay(ctx context.Context, tenantID, staffID string, input CreateStaffPayInput) (*payrollrepo.StaffPay, error)
	UpdateStaffPay(ctx context.Context, tenantID, staffID, payID string, input UpdateStaffPayInput) (*payrollrepo.StaffPay, error)
	ListStaffPayHistory(ctx context.Context, tenantID, staffID string, page store.Page) (store.ResultList[payrollrepo.StaffPay], error)
	ToggleStaffPayStatus(ctx context.Context, tenantID, staffID, payID string) (*payrollrepo.StaffPay, error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

type staffPayService struct {
	repo payrollrepo.StaffPayRepo
}

func NewStaffPayService(r payrollrepo.StaffPayRepo) StaffPayService {
	return &staffPayService{repo: r}
}

func (s *staffPayService) validatePayType(pt payrollrepo.PayType) bool {
	return pt == payrollrepo.PayTypeSalary || pt == payrollrepo.PayTypeHourly || pt == payrollrepo.PayTypeSalaryPlusCommission || pt == payrollrepo.PayTypeHourlyPlusCommission
}

func (s *staffPayService) validatePayFrequency(pf payrollrepo.PayFrequency) bool {
	return pf == payrollrepo.PayFrequencyMonthly || pf == payrollrepo.PayFrequencyBiweekly || pf == payrollrepo.PayFrequencyWeekly || pf == payrollrepo.PayFrequencyDaily || pf == payrollrepo.PayFrequencyPerService
}

func (s *staffPayService) GetActiveStaffPay(ctx context.Context, tenantID, staffID string) (*payrollrepo.StaffPay, error) {
	if tenantID == "" || staffID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.GetActive", "tenantID & staffID required")
	}

	pay, err := s.repo.GetActive(ctx, tenantID, staffID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "StaffPayService.GetActive", "No active pay plan found")
		}
		return nil, errors.Wrap(err, errors.Internal, "StaffPayService.GetActive", "Failed to get active pay")
	}
	return pay, nil
}

func (s *staffPayService) CreateStaffPay(ctx context.Context, tenantID, staffID string, input CreateStaffPayInput) (*payrollrepo.StaffPay, error) {
	if tenantID == "" || staffID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.Create", "tenantID & staffID required")
	}

	if !s.validatePayType(input.PayType) || !s.validatePayFrequency(input.PayFrequency) {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.Create", "Invalid pay_type or pay_frequency")
	}

	if input.Amount <= 0 {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.Create", "amount must be > 0")
	}

	if input.PayType == payrollrepo.PayTypeSalaryPlusCommission || input.PayType == payrollrepo.PayTypeHourlyPlusCommission {
		if input.CommissionRate == nil || *input.CommissionRate < 0 || *input.CommissionRate > 1 {
			return nil, errors.New(errors.InvalidArgument, "StaffPayService.Create", "commission_rate must be between 0 and 1")
		}
	}

	start, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.Create", "Invalid start_date format")
	}

	var end *time.Time
	if input.EndDate != nil {
		pEnd, err := time.Parse("2006-01-02", *input.EndDate)
		if err != nil {
			return nil, errors.New(errors.InvalidArgument, "StaffPayService.Create", "Invalid end_date format")
		}
		if pEnd.Before(start) {
			return nil, errors.New(errors.InvalidArgument, "StaffPayService.Create", "end_date must be >= start_date")
		}
		end = &pEnd
	}

	// Deactivate current active plan, setting its effective_to to start-1day (so they don't overlap)
	// overlapping constraint is: daterange(effective_from, effective_to, '[)') WITH &&
	// So effective_to is exclusive. If we set effective_to = start, then [old_start, start) and [start, new_end) do NOT overlap.
	if err := s.repo.DeactivateActivePlan(ctx, tenantID, staffID, start); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "StaffPayService.Create", "Failed to deactivate current plan")
	}

	pay := &payrollrepo.StaffPay{
		TenantID:       tenantID,
		StaffID:        staffID,
		PayType:        input.PayType,
		PayFrequency:   input.PayFrequency,
		Amount:         input.Amount,
		CommissionRate: input.CommissionRate,
		StartDate:      start,
		EndDate:        end,
		IsActive:       true, // Default active
	}

	if err := s.repo.Create(ctx, pay); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "StaffPayService.Create", "Failed to create staff pay plan")
	}

	return pay, nil
}

func (s *staffPayService) UpdateStaffPay(ctx context.Context, tenantID, staffID, payID string, input UpdateStaffPayInput) (*payrollrepo.StaffPay, error) {
	if input.PayType != nil && !s.validatePayType(*input.PayType) {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.Update", "Invalid pay_type")
	}
	if input.PayFrequency != nil && !s.validatePayFrequency(*input.PayFrequency) {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.Update", "Invalid pay_frequency")
	}
	if input.Amount != nil && *input.Amount <= 0 {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.Update", "amount must be > 0")
	}

	// Fetch current to validate cross-dependencies
	current, err := s.repo.GetByID(ctx, tenantID, staffID, payID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "StaffPayService.Update", "Pay plan not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "StaffPayService.Update", "Failed to fetch plan")
	}

	payTypeToEvaluate := current.PayType
	if input.PayType != nil {
		payTypeToEvaluate = *input.PayType
	}

	commToEvaluate := current.CommissionRate
	if input.CommissionRate != nil {
		commToEvaluate = input.CommissionRate
	}

	if payTypeToEvaluate == payrollrepo.PayTypeSalaryPlusCommission || payTypeToEvaluate == payrollrepo.PayTypeHourlyPlusCommission {
		if commToEvaluate == nil || *commToEvaluate < 0 || *commToEvaluate > 1 {
			return nil, errors.New(errors.InvalidArgument, "StaffPayService.Update", "commission_rate must be between 0 and 1")
		}
	}

	pay, err := s.repo.Update(ctx, tenantID, staffID, payID, payrollrepo.UpdateFields{
		PayType:        input.PayType,
		PayFrequency:   input.PayFrequency,
		Amount:         input.Amount,
		CommissionRate: input.CommissionRate,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "StaffPayService.Update", "Failed to update staff pay plan")
	}

	return pay, nil
}

func (s *staffPayService) ListStaffPayHistory(ctx context.Context, tenantID, staffID string, page store.Page) (store.ResultList[payrollrepo.StaffPay], error) {
	if tenantID == "" || staffID == "" {
		return store.ResultList[payrollrepo.StaffPay]{}, errors.New(errors.InvalidArgument, "StaffPayService.List", "tenantID & staffID required")
	}

	result, err := s.repo.List(ctx, tenantID, staffID, page)
	if err != nil {
		return store.ResultList[payrollrepo.StaffPay]{}, errors.Wrap(err, errors.Internal, "StaffPayService.List", "Failed to list staff pay history")
	}

	return result, err
}

func (s *staffPayService) ToggleStaffPayStatus(ctx context.Context, tenantID, staffID, payID string) (*payrollrepo.StaffPay, error) {
	if tenantID == "" || payID == "" || staffID == "" {
		return nil, errors.New(errors.InvalidArgument, "StaffPayService.ToggleStatus", "tenantID, staffID, & payID required")
	}

	current, err := s.repo.GetByID(ctx, tenantID, staffID, payID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "StaffPayService.ToggleStatus", "Pay plan not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "StaffPayService.ToggleStatus", "Failed to fetch plan")
	}

	if !current.IsActive {
		// Activating this one -> deactivate others
		// We set effective_to of the other active to "today" to avoid overlap,
		// and set this one's effective_from to "today" if it's in the past (to prevent overlap) ?
		// Actually, let's keep it simple: deactivate old, and set start of this one to today so it takes over.
		now := time.Now().Truncate(24 * time.Hour)
		_ = s.repo.DeactivateActivePlan(ctx, tenantID, staffID, now)

		t := true
		return s.repo.Update(ctx, tenantID, staffID, payID, payrollrepo.UpdateFields{
			IsActive:  &t,
			StartDate: &now,
		})
	}

	// Deactivating
	f := false
	return s.repo.Update(ctx, tenantID, staffID, payID, payrollrepo.UpdateFields{IsActive: &f})
}
