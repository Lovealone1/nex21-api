package payrollservice

import (
	"context"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

type CreateRunInput struct {
	LocationID  *string                    `json:"location_id,omitempty"`
	Frequency   payrolldomain.RunFrequency `json:"frequency"`
	PeriodStart string                     `json:"period_start"` // YYYY-MM-DD
	PeriodEnd   string                     `json:"period_end"`   // YYYY-MM-DD
	PayDate     string                     `json:"pay_date"`     // YYYY-MM-DD
	Currency    string                     `json:"currency,omitempty"`
	Notes       *string                    `json:"notes,omitempty"`
}

type UpdateRunInput struct {
	LocationID  *string                     `json:"location_id,omitempty"`
	Frequency   *payrolldomain.RunFrequency `json:"frequency,omitempty"`
	PeriodStart *string                     `json:"period_start,omitempty"`
	PeriodEnd   *string                     `json:"period_end,omitempty"`
	PayDate     *string                     `json:"pay_date,omitempty"`
	Currency    *string                     `json:"currency,omitempty"`
	Notes       *string                     `json:"notes,omitempty"`
}

type UpdateRunStatusInput struct {
	Status payrolldomain.RunStatus `json:"status"`
}

type PayrollRunService interface {
	CreateRun(ctx context.Context, tenantID string, input CreateRunInput) (*payrollrepo.PayrollRun, error)
	GetRunByID(ctx context.Context, tenantID, runID string) (*payrollrepo.PayrollRun, error)
	UpdateRun(ctx context.Context, tenantID, runID string, input UpdateRunInput) (*payrollrepo.PayrollRun, error)
	DeleteRun(ctx context.Context, tenantID, runID string) error
	ListRuns(ctx context.Context, tenantID string, filters payrollrepo.RunFilters, page store.Page) (store.ResultList[payrollrepo.PayrollRun], error)
	UpdateRunStatus(ctx context.Context, tenantID, runID string, input UpdateRunStatusInput) (*payrollrepo.PayrollRun, error)
}

type payrollRunService struct {
	repo payrollrepo.PayrollRunRepo
}

func NewPayrollRunService(r payrollrepo.PayrollRunRepo) PayrollRunService {
	return &payrollRunService{repo: r}
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func (s *payrollRunService) CreateRun(ctx context.Context, tenantID string, input CreateRunInput) (*payrollrepo.PayrollRun, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Create", "tenantID required")
	}

	start, err := parseDate(input.PeriodStart)
	if err != nil {
		return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Create", "Invalid period_start format (use YYYY-MM-DD)")
	}
	end, err := parseDate(input.PeriodEnd)
	if err != nil {
		return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Create", "Invalid period_end format")
	}
	if end.Before(start) {
		return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Create", "period_end must be >= period_start")
	}
	payDate, err := parseDate(input.PayDate)
	if err != nil {
		return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Create", "Invalid pay_date format")
	}

	currency := "COP"
	if input.Currency != "" {
		currency = input.Currency
	}

	run := &payrollrepo.PayrollRun{
		TenantID:    tenantID,
		LocationID:  input.LocationID,
		Frequency:   input.Frequency,
		PeriodStart: start,
		PeriodEnd:   end,
		PayDate:     payDate,
		Status:      payrolldomain.StatusDraft, // Default
		Total:       0,
		Currency:    currency,
		Notes:       input.Notes,
	}

	if err := s.repo.Create(ctx, run); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "PayrollRunService.Create", "Failed to create run (check unique constraint)")
	}

	return run, nil
}

func (s *payrollRunService) GetRunByID(ctx context.Context, tenantID, runID string) (*payrollrepo.PayrollRun, error) {
	run, err := s.repo.GetByID(ctx, tenantID, runID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "PayrollRunService.Get", "Run not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "PayrollRunService.Get", "Failed to fetch run")
	}
	return run, nil
}

func (s *payrollRunService) UpdateRun(ctx context.Context, tenantID, runID string, input UpdateRunInput) (*payrollrepo.PayrollRun, error) {
	current, err := s.GetRunByID(ctx, tenantID, runID)
	if err != nil {
		return nil, err
	}

	if current.Status != payrolldomain.StatusDraft {
		return nil, errors.New(errors.FailedPrecondition, "PayrollRunService.Update", "Runs can only be edited while in 'draft' status")
	}

	fields := payrollrepo.UpdateRunFields{
		LocationID: input.LocationID,
		Frequency:  input.Frequency,
		Currency:   input.Currency,
		Notes:      input.Notes,
	}

	var parsedStart, parsedEnd time.Time
	if input.PeriodStart != nil {
		parsedStart, err = parseDate(*input.PeriodStart)
		if err != nil {
			return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Update", "Invalid period_start")
		}
		fields.PeriodStart = &parsedStart
	}
	if input.PeriodEnd != nil {
		parsedEnd, err = parseDate(*input.PeriodEnd)
		if err != nil {
			return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Update", "Invalid period_end")
		}
		fields.PeriodEnd = &parsedEnd
	}

	// Validate date rules
	evalStart := current.PeriodStart
	if fields.PeriodStart != nil {
		evalStart = *fields.PeriodStart
	}
	evalEnd := current.PeriodEnd
	if fields.PeriodEnd != nil {
		evalEnd = *fields.PeriodEnd
	}
	if evalEnd.Before(evalStart) {
		return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Update", "period_end must be >= period_start")
	}

	if input.PayDate != nil {
		pd, err := parseDate(*input.PayDate)
		if err != nil {
			return nil, errors.New(errors.InvalidArgument, "PayrollRunService.Update", "Invalid pay_date")
		}
		fields.PayDate = &pd
	}

	run, err := s.repo.Update(ctx, tenantID, runID, fields)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "PayrollRunService.Update", "Failed to update run")
	}

	return run, nil
}

func (s *payrollRunService) DeleteRun(ctx context.Context, tenantID, runID string) error {
	current, err := s.GetRunByID(ctx, tenantID, runID)
	if err != nil {
		return err
	}

	if current.Status != payrolldomain.StatusDraft {
		return errors.New(errors.FailedPrecondition, "PayrollRunService.Delete", "Only 'draft' runs can be deleted. Consider cancelling instead.")
	}

	if err := s.repo.Delete(ctx, tenantID, runID); err != nil {
		return errors.Wrap(err, errors.Internal, "PayrollRunService.Delete", "Failed to delete run")
	}
	return nil
}

func (s *payrollRunService) ListRuns(ctx context.Context, tenantID string, filters payrollrepo.RunFilters, page store.Page) (store.ResultList[payrollrepo.PayrollRun], error) {
	res, err := s.repo.List(ctx, tenantID, filters, page)
	if err != nil {
		return store.ResultList[payrollrepo.PayrollRun]{}, errors.Wrap(err, errors.Internal, "PayrollRunService.List", "Failed to list runs")
	}
	return res, nil
}

func (s *payrollRunService) UpdateRunStatus(ctx context.Context, tenantID, runID string, input UpdateRunStatusInput) (*payrollrepo.PayrollRun, error) {
	current, err := s.GetRunByID(ctx, tenantID, runID)
	if err != nil {
		return nil, err
	}

	if err := validateStatusTransition(current.Status, input.Status); err != nil {
		return nil, err
	}

	run, err := s.repo.Update(ctx, tenantID, runID, payrollrepo.UpdateRunFields{Status: &input.Status})
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "PayrollRunService.UpdateStatus", "Failed to update run status")
	}

	return run, nil
}

func validateStatusTransition(current, next payrolldomain.RunStatus) error {
	if current == next {
		return nil // No op
	}

	switch current {
	case payrolldomain.StatusDraft:
		if next == payrolldomain.StatusApproved || next == payrolldomain.StatusCancelled {
			return nil
		}
	case payrolldomain.StatusApproved:
		if next == payrolldomain.StatusPaid || next == payrolldomain.StatusCancelled || next == payrolldomain.StatusDraft {
			// Often allowed to revert to draft if mistake found
			return nil
		}
	case payrolldomain.StatusPaid:
		// Disallow paid -> anything
		return errors.New(errors.FailedPrecondition, "PayrollRunService", "A 'paid' payroll run cannot be modified")
	case payrolldomain.StatusCancelled:
		// Disallow cancelled -> anything
		return errors.New(errors.FailedPrecondition, "PayrollRunService", "A 'cancelled' payroll run cannot be un-cancelled")
	}

	return errors.New(errors.InvalidArgument, "PayrollRunService", "Invalid status transition")
}
