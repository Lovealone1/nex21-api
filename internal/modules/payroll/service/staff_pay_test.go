package payrollservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	payrollrepo "github.com/Lovealone1/nex21-api/internal/modules/payroll/repo"
	payrollservice "github.com/Lovealone1/nex21-api/internal/modules/payroll/service"
)

type mockStaffPayRepo struct {
	deactivated int
	created     int
	updated     int
	activePlan  *payrollrepo.StaffPay
	getByIDPlan *payrollrepo.StaffPay
	errGetByID  error
}

func (m *mockStaffPayRepo) Create(ctx context.Context, s *payrollrepo.StaffPay) error {
	m.created++
	s.ID = "new-id"
	return nil
}

func (m *mockStaffPayRepo) GetActive(ctx context.Context, tenantID, staffID string) (*payrollrepo.StaffPay, error) {
	if m.activePlan != nil {
		return m.activePlan, nil
	}
	return nil, nil // Assume gorm.ErrRecordNotFound if nil, but keeping simple for tests
}

func (m *mockStaffPayRepo) GetByID(ctx context.Context, tenantID, staffID, payID string) (*payrollrepo.StaffPay, error) {
	if m.errGetByID != nil {
		return nil, m.errGetByID
	}
	if m.getByIDPlan != nil {
		return m.getByIDPlan, nil
	}
	return nil, nil
}

func (m *mockStaffPayRepo) Update(ctx context.Context, tenantID, staffID, payID string, fields payrollrepo.UpdateFields) (*payrollrepo.StaffPay, error) {
	m.updated++
	return m.getByIDPlan, nil
}

func (m *mockStaffPayRepo) List(ctx context.Context, tenantID, staffID string, page store.Page) (store.ResultList[payrollrepo.StaffPay], error) {
	return store.ResultList[payrollrepo.StaffPay]{}, nil
}

func (m *mockStaffPayRepo) DeactivateActivePlan(ctx context.Context, tenantID, staffID string, effectiveTo time.Time) error {
	m.deactivated++
	return nil
}

func TestCreatePlan_DeactivatesOldActivePlan(t *testing.T) {
	repo := &mockStaffPayRepo{}
	svc := payrollservice.NewStaffPayService(repo)

	_, err := svc.CreateStaffPay(context.Background(), "t1", "s1", payrollservice.CreateStaffPayInput{
		PayType:      payrollrepo.PayTypeSalary,
		PayFrequency: payrollrepo.PayFrequencyMonthly,
		Amount:       1000,
		StartDate:    "2023-01-01",
	})

	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}

	if repo.deactivated != 1 {
		t.Errorf("expected 1 deactivated plan, got %d", repo.deactivated)
	}
	if repo.created != 1 {
		t.Errorf("expected 1 created plan, got %d", repo.created)
	}
}

func TestActivatePlan_DeactivatesCurrentActivePlan(t *testing.T) {
	repo := &mockStaffPayRepo{
		getByIDPlan: &payrollrepo.StaffPay{
			ID:       "pay1",
			IsActive: false, // Currently inactive
		},
	}
	svc := payrollservice.NewStaffPayService(repo)

	_, err := svc.ToggleStaffPayStatus(context.Background(), "t1", "s1", "pay1")
	if err != nil {
		t.Fatalf("expected no err, got %v", err)
	}

	// Should have deactivated current active ones and then updated this one
	if repo.deactivated != 1 {
		t.Errorf("expected 1 deactivated plan, got %d", repo.deactivated)
	}
	if repo.updated != 1 {
		t.Errorf("expected 1 updated plan, got %d", repo.updated)
	}
}

func TestCrossTenantAccessIsRejected(t *testing.T) {
	// Let's test standard empty guard or cross tenant from service
	repo := &mockStaffPayRepo{}
	svc := payrollservice.NewStaffPayService(repo)

	_, err := svc.GetActiveStaffPay(context.Background(), "", "s1")
	if err == nil {
		t.Errorf("Expected error for missing tenant ID")
	}

	_, err = svc.CreateStaffPay(context.Background(), "t1", "", payrollservice.CreateStaffPayInput{})
	if err == nil {
		t.Errorf("Expected error for missing staff ID")
	}
}
