package payrollservice_test

import (
	"context"
	"testing"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrolldomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	payrollservice "github.com/Lovealone1/nex21-api/internal/modules/payroll/service"
)

// -----------------------------------------------------------------------------
// MOCKS
// -----------------------------------------------------------------------------

type mockPayrollItemRepo struct {
	getItem *payrollrepo.PayrollItem
	created int
	updated int
	deleted int
}

func (m *mockPayrollItemRepo) Create(ctx context.Context, item *payrollrepo.PayrollItem) error {
	m.created++
	item.ID = "new-item-id"
	return nil
}

func (m *mockPayrollItemRepo) GetByID(ctx context.Context, tenantID, runID, itemID string) (*payrollrepo.PayrollItem, error) {
	return m.getItem, nil
}

func (m *mockPayrollItemRepo) Update(ctx context.Context, tenantID, runID, itemID string, fields payrollrepo.UpdateItemFields) (*payrollrepo.PayrollItem, error) {
	m.updated++
	return m.getItem, nil
}

func (m *mockPayrollItemRepo) Delete(ctx context.Context, tenantID, runID, itemID string) error {
	m.deleted++
	return nil
}

func (m *mockPayrollItemRepo) List(ctx context.Context, tenantID, runID string, filters payrollrepo.ItemFilters, page store.Page) (store.ResultList[payrollrepo.PayrollItem], error) {
	return store.ResultList[payrollrepo.PayrollItem]{}, nil
}

// -----------------------------------------------------------------------------
// TESTS
// -----------------------------------------------------------------------------

func TestItemMutationOnlyWhenRunIsDraft(t *testing.T) {
	runRepoAppr := &mockPayrollRunRepo{
		getByIDPlan: &payrollrepo.PayrollRun{ID: "run-appr", Status: payrolldomain.StatusApproved},
	}
	itemRepo := &mockPayrollItemRepo{}

	svcAppr := payrollservice.NewPayrollItemService(itemRepo, runRepoAppr)

	// Since run is approved, creating an item should fail
	if _, err := svcAppr.CreateItem(context.Background(), "t1", "run-appr", payrollservice.CreateItemInput{Amount: 100}); err == nil {
		t.Errorf("expected err when creating item in approved run")
	}

	runRepoDraft := &mockPayrollRunRepo{
		getByIDPlan: &payrollrepo.PayrollRun{ID: "run-draft", Status: payrolldomain.StatusDraft},
	}
	svcDraft := payrollservice.NewPayrollItemService(itemRepo, runRepoDraft)

	// Since run is draft, creating an item should succeed
	if _, err := svcDraft.CreateItem(context.Background(), "t1", "run-draft", payrollservice.CreateItemInput{Amount: 100}); err != nil {
		t.Errorf("expected no err when creating item in draft run, got %v", err)
	}
	if itemRepo.created != 1 {
		t.Errorf("expected 1 item created")
	}
}
