package service_test

import (
	"context"
	"testing"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/finance/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/finance/service"
	"github.com/jackc/pgx/v5/pgconn"
)

// -----------------------------------------------------------------------------
// MOCKS
// -----------------------------------------------------------------------------

type mockAccountRepo struct {
	getByIDPlan   *repo.Account
	errGetByID    error
	errDelete     error
	deleteCalled  bool
	toggleCalled  bool
	setDefaultCnt int
}

func (m *mockAccountRepo) Create(ctx context.Context, acc *repo.Account) error {
	acc.ID = "new-account-id"
	return nil
}

func (m *mockAccountRepo) GetByID(ctx context.Context, tenantID, id string) (*repo.Account, error) {
	if m.errGetByID != nil {
		return nil, m.errGetByID
	}
	return m.getByIDPlan, nil
}

func (m *mockAccountRepo) GetDefault(ctx context.Context, tenantID string) (*repo.Account, error) {
	return nil, nil
}

func (m *mockAccountRepo) Update(ctx context.Context, tenantID, id string, fields repo.UpdateAccountFields) (*repo.Account, error) {
	return m.getByIDPlan, nil
}

func (m *mockAccountRepo) Delete(ctx context.Context, tenantID, id string) error {
	m.deleteCalled = true
	return m.errDelete
}

func (m *mockAccountRepo) List(ctx context.Context, tenantID string, filters repo.AccountFilters, page store.Page) (store.ResultList[repo.Account], error) {
	return store.ResultList[repo.Account]{}, nil
}

func (m *mockAccountRepo) SetDefaultTx(ctx context.Context, tenantID, accountID string) error {
	m.setDefaultCnt++
	return nil
}

func (m *mockAccountRepo) ToggleStatus(ctx context.Context, tenantID, accountID string, isActive bool) (*repo.Account, error) {
	m.toggleCalled = true
	m.getByIDPlan.IsActive = isActive
	if !isActive {
		m.getByIDPlan.IsDefault = false
	}
	return m.getByIDPlan, nil
}

// -----------------------------------------------------------------------------
// TESTS
// -----------------------------------------------------------------------------

func TestSetDefaultWhenInactiveFails(t *testing.T) {
	repo := &mockAccountRepo{
		getByIDPlan: &repo.Account{ID: "acc1", IsActive: false},
	}
	svc := service.NewAccountService(repo)

	_, err := svc.SetDefaultAccount(context.Background(), "tenant1", "acc1")
	if err == nil {
		t.Errorf("expected error when setting inactive account as default")
	}
	if repo.setDefaultCnt != 0 {
		t.Errorf("SetDefaultTx should not have been called")
	}
}

func TestDeleteFallbackToSoftDelete(t *testing.T) {
	fkErr := &pgconn.PgError{Code: "23503"}

	repo := &mockAccountRepo{
		getByIDPlan: &repo.Account{ID: "acc1", IsActive: true},
		errDelete:   fkErr, // simulate postgres FK rejection
	}
	svc := service.NewAccountService(repo)

	err := svc.DeleteAccount(context.Background(), "tenant1", "acc1")
	if err != nil {
		t.Errorf("expected DeleteAccount to swallow FK error and fallback to soft delete, got err: %v", err)
	}
	if !repo.deleteCalled {
		t.Errorf("expected Hard delete to be attempted")
	}
	if !repo.toggleCalled {
		t.Errorf("expected toggleStatus to be called as a fallback")
	}
	if repo.getByIDPlan.IsActive != false {
		t.Errorf("expected account to be marked inactive by toggleStatus")
	}
}

func TestDeactivatingDefaultUnsetsDefault(t *testing.T) {
	repo := &mockAccountRepo{
		getByIDPlan: &repo.Account{ID: "acc1", IsActive: true, IsDefault: true},
	}
	svc := service.NewAccountService(repo)

	f := false
	_, err := svc.UpdateAccount(context.Background(), "t1", "acc1", service.UpdateAccountInput{IsActive: &f})
	if err != nil {
		t.Errorf("expected successful update, got: %v", err)
	}

	if !repo.toggleCalled {
		t.Errorf("expected toggle status to be invoked to natively clear the default bit")
	}
	if repo.getByIDPlan.IsDefault != false {
		t.Errorf("expected IsDefault to be natively flipped to false")
	}
}

func TestCrossTenantRejection(t *testing.T) {
	repo := &mockAccountRepo{}
	svc := service.NewAccountService(repo)

	// Missing tenant
	if _, err := svc.CreateAccount(context.Background(), "", service.CreateAccountInput{Name: "test"}); err == nil {
		t.Errorf("expected error when tenant ID is missing")
	}
}
