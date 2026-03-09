package payrollservice

import (
	"context"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

type CreateItemInput struct {
	StaffID       string                    `json:"staff_id"`
	LineType      payrolldomain.LineType    `json:"line_type"`
	Concept       payrolldomain.ItemConcept `json:"concept"`
	Amount        float64                   `json:"amount"`
	AppointmentID *string                   `json:"appointment_id,omitempty"`
	ServiceID     *string                   `json:"service_id,omitempty"`
	Notes         *string                   `json:"notes,omitempty"`
}

type UpdateItemInput struct {
	LineType      *payrolldomain.LineType    `json:"line_type,omitempty"`
	Concept       *payrolldomain.ItemConcept `json:"concept,omitempty"`
	Amount        *float64                   `json:"amount,omitempty"`
	AppointmentID *string                    `json:"appointment_id,omitempty"`
	ServiceID     *string                    `json:"service_id,omitempty"`
	Notes         *string                    `json:"notes,omitempty"`
}

type PayrollItemService interface {
	CreateItem(ctx context.Context, tenantID, runID string, input CreateItemInput) (*payrollrepo.PayrollItem, error)
	GetItemByID(ctx context.Context, tenantID, runID, itemID string) (*payrollrepo.PayrollItem, error)
	UpdateItem(ctx context.Context, tenantID, runID, itemID string, input UpdateItemInput) (*payrollrepo.PayrollItem, error)
	DeleteItem(ctx context.Context, tenantID, runID, itemID string) error
	ListItems(ctx context.Context, tenantID, runID string, filters payrollrepo.ItemFilters, page store.Page) (store.ResultList[payrollrepo.PayrollItem], error)
}

type payrollItemService struct {
	itemRepo payrollrepo.PayrollItemRepo
	runRepo  payrollrepo.PayrollRunRepo
}

func NewPayrollItemService(iRepo payrollrepo.PayrollItemRepo, rRepo payrollrepo.PayrollRunRepo) PayrollItemService {
	return &payrollItemService{itemRepo: iRepo, runRepo: rRepo}
}

func (s *payrollItemService) ensureRunIsDraft(ctx context.Context, tenantID, runID string) error {
	run, err := s.runRepo.GetByID(ctx, tenantID, runID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "PayrollItemService", "Payroll run not found")
		}
		return errors.Wrap(err, errors.Internal, "PayrollItemService", "Failed to fetch payroll run")
	}

	if run.Status != payrolldomain.StatusDraft {
		return errors.New(errors.FailedPrecondition, "PayrollItemService", "Items can only be modified when the payroll run is in 'draft' status")
	}

	return nil
}

func (s *payrollItemService) CreateItem(ctx context.Context, tenantID, runID string, input CreateItemInput) (*payrollrepo.PayrollItem, error) {
	if err := s.ensureRunIsDraft(ctx, tenantID, runID); err != nil {
		return nil, err
	}

	if input.Amount <= 0 {
		return nil, errors.New(errors.InvalidArgument, "PayrollItemService.Create", "amount must be > 0")
	}

	item := &payrollrepo.PayrollItem{
		TenantID:      tenantID,
		PayrollRunID:  runID,
		StaffID:       input.StaffID,
		LineType:      input.LineType,
		Concept:       input.Concept,
		Amount:        input.Amount,
		AppointmentID: input.AppointmentID,
		ServiceID:     input.ServiceID,
		Notes:         input.Notes,
	}

	if err := s.itemRepo.Create(ctx, item); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "PayrollItemService.Create", "Failed to create item (check unique constraint)")
	}

	return item, nil
}

func (s *payrollItemService) GetItemByID(ctx context.Context, tenantID, runID, itemID string) (*payrollrepo.PayrollItem, error) {
	item, err := s.itemRepo.GetByID(ctx, tenantID, runID, itemID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "PayrollItemService.Get", "Item not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "PayrollItemService.Get", "Failed to fetch item")
	}
	return item, nil
}

func (s *payrollItemService) UpdateItem(ctx context.Context, tenantID, runID, itemID string, input UpdateItemInput) (*payrollrepo.PayrollItem, error) {
	if err := s.ensureRunIsDraft(ctx, tenantID, runID); err != nil {
		return nil, err
	}

	if input.Amount != nil && *input.Amount <= 0 {
		return nil, errors.New(errors.InvalidArgument, "PayrollItemService.Update", "amount must be > 0")
	}

	fields := payrollrepo.UpdateItemFields{
		LineType:      input.LineType,
		Concept:       input.Concept,
		Amount:        input.Amount,
		AppointmentID: input.AppointmentID,
		ServiceID:     input.ServiceID,
		Notes:         input.Notes,
	}

	item, err := s.itemRepo.Update(ctx, tenantID, runID, itemID, fields)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "PayrollItemService.Update", "Item not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "PayrollItemService.Update", "Failed to update item")
	}

	return item, nil
}

func (s *payrollItemService) DeleteItem(ctx context.Context, tenantID, runID, itemID string) error {
	if err := s.ensureRunIsDraft(ctx, tenantID, runID); err != nil {
		return err
	}

	if err := s.itemRepo.Delete(ctx, tenantID, runID, itemID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "PayrollItemService.Delete", "Item not found")
		}
		return errors.Wrap(err, errors.Internal, "PayrollItemService.Delete", "Failed to delete item")
	}

	return nil
}

func (s *payrollItemService) ListItems(ctx context.Context, tenantID, runID string, filters payrollrepo.ItemFilters, page store.Page) (store.ResultList[payrollrepo.PayrollItem], error) {
	res, err := s.itemRepo.List(ctx, tenantID, runID, filters, page)
	if err != nil {
		return store.ResultList[payrollrepo.PayrollItem]{}, errors.Wrap(err, errors.Internal, "PayrollItemService.List", "Failed to list items")
	}
	return res, nil
}
