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

type mockPayrollRunRepo struct {
	getByIDPlan *payrollrepo.PayrollRun
	errGetByID  error
	created     int
	updated     int
	deleted     int
}

func (m *mockPayrollRunRepo) Create(ctx context.Context, r *payrollrepo.PayrollRun) error {
	m.created++
	r.ID = "new-run-id"
	return nil
}

func (m *mockPayrollRunRepo) GetByID(ctx context.Context, tenantID, runID string) (*payrollrepo.PayrollRun, error) {
	if m.errGetByID != nil {
		return nil, m.errGetByID
	}
	if m.getByIDPlan != nil {
		return m.getByIDPlan, nil
	}
	return nil, nil
}

func (m *mockPayrollRunRepo) Update(ctx context.Context, tenantID, runID string, fields payrollrepo.UpdateRunFields) (*payrollrepo.PayrollRun, error) {
	m.updated++
	if fields.Status != nil && m.getByIDPlan != nil {
		m.getByIDPlan.Status = *fields.Status
	}
	return m.getByIDPlan, nil
}

func (m *mockPayrollRunRepo) Delete(ctx context.Context, tenantID, runID string) error {
	m.deleted++
	return nil
}

func (m *mockPayrollRunRepo) List(ctx context.Context, tenantID string, filters payrollrepo.RunFilters, page store.Page) (store.ResultList[payrollrepo.PayrollRun], error) {
	return store.ResultList[payrollrepo.PayrollRun]{}, nil
}

// -----------------------------------------------------------------------------
// TESTS
// -----------------------------------------------------------------------------

func TestRunStatusTransitions(t *testing.T) {
	repo := &mockPayrollRunRepo{
		getByIDPlan: &payrollrepo.PayrollRun{
			ID:     "run-draft",
			Status: payrolldomain.StatusDraft,
		},
	}
	svc := payrollservice.NewPayrollRunService(repo)

	// Draft -> Approved : Allowed
	if _, err := svc.UpdateRunStatus(context.Background(), "t1", "run-draft", payrollservice.UpdateRunStatusInput{Status: payrolldomain.StatusApproved}); err != nil {
		t.Errorf("expected no err for draft -> approved, got %v", err)
	}

	// Draft -> Paid : Not allowed
	repo.getByIDPlan.Status = payrolldomain.StatusDraft
	if _, err := svc.UpdateRunStatus(context.Background(), "t1", "run-draft", payrollservice.UpdateRunStatusInput{Status: payrolldomain.StatusPaid}); err == nil {
		t.Errorf("expected err for draft -> paid, got nil")
	}

	// Approved -> Paid : Allowed
	repo.getByIDPlan.Status = payrolldomain.StatusApproved
	if _, err := svc.UpdateRunStatus(context.Background(), "t1", "run-appr", payrollservice.UpdateRunStatusInput{Status: payrolldomain.StatusPaid}); err != nil {
		t.Errorf("expected no err for approved -> paid, got %v", err)
	}

	// Paid -> Draft (anything): Not allowed
	repo.getByIDPlan.Status = payrolldomain.StatusPaid
	if _, err := svc.UpdateRunStatus(context.Background(), "t1", "run-paid", payrollservice.UpdateRunStatusInput{Status: payrolldomain.StatusDraft}); err == nil {
		t.Errorf("expected err for paid -> draft, got nil")
	}
}

func TestMutationOnlyWhenDraft(t *testing.T) {
	repo := &mockPayrollRunRepo{
		getByIDPlan: &payrollrepo.PayrollRun{
			ID:     "run-appr",
			Status: payrolldomain.StatusApproved,
		},
	}
	svc := payrollservice.NewPayrollRunService(repo)

	// Trying to update non-draft run
	if _, err := svc.UpdateRun(context.Background(), "t1", "run-appr", payrollservice.UpdateRunInput{}); err == nil {
		t.Errorf("expected err when updating approved run")
	}

	// Trying to delete non-draft run
	if err := svc.DeleteRun(context.Background(), "t1", "run-appr"); err == nil {
		t.Errorf("expected err when deleting approved run")
	}
}

func TestRunCrossTenantReject(t *testing.T) {
	repo := &mockPayrollRunRepo{}
	svc := payrollservice.NewPayrollRunService(repo)

	if _, err := svc.CreateRun(context.Background(), "", payrollservice.CreateRunInput{}); err == nil {
		t.Errorf("expected err for missing tenantID")
	}
}
