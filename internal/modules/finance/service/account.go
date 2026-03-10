package service

import (
	"context"
	"errors"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	domain "github.com/Lovealone1/nex21-api/internal/modules/finance/domain"
	repo "github.com/Lovealone1/nex21-api/internal/modules/finance/repo"
	apperr "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type CreateAccountInput struct {
	Name        string             `json:"name"`
	Code        *string            `json:"code,omitempty"`
	AccountType domain.AccountType `json:"account_type"`
	Currency    string             `json:"currency,omitempty"`
	IsActive    *bool              `json:"is_active,omitempty"`
	IsDefault   *bool              `json:"is_default,omitempty"`
	Provider    *string            `json:"provider,omitempty"`
	Notes       *string            `json:"notes,omitempty"`
}

type UpdateAccountInput struct {
	Name        *string             `json:"name,omitempty"`
	Code        *string             `json:"code,omitempty"`
	AccountType *domain.AccountType `json:"account_type,omitempty"`
	Currency    *string             `json:"currency,omitempty"`
	IsActive    *bool               `json:"is_active,omitempty"`
	Provider    *string             `json:"provider,omitempty"`
	Notes       *string             `json:"notes,omitempty"`
}

type AccountService interface {
	CreateAccount(ctx context.Context, tenantID string, input CreateAccountInput) (*repo.Account, error)
	GetAccountByID(ctx context.Context, tenantID, accountID string) (*repo.Account, error)
	GetDefaultAccount(ctx context.Context, tenantID string) (*repo.Account, error)
	UpdateAccount(ctx context.Context, tenantID, accountID string, input UpdateAccountInput) (*repo.Account, error)
	DeleteAccount(ctx context.Context, tenantID, accountID string) error
	ListAccounts(ctx context.Context, tenantID string, filters repo.AccountFilters, page store.Page) (store.ResultList[repo.Account], error)

	SetDefaultAccount(ctx context.Context, tenantID, accountID string) (*repo.Account, error)
	ToggleAccountStatus(ctx context.Context, tenantID, accountID string, isActive bool) (*repo.Account, error)
}

type accountService struct {
	repo repo.AccountRepo
}

func NewAccountService(repo repo.AccountRepo) AccountService {
	return &accountService{repo: repo}
}

func (s *accountService) CreateAccount(ctx context.Context, tenantID string, input CreateAccountInput) (*repo.Account, error) {
	if tenantID == "" {
		return nil, apperr.New(apperr.InvalidArgument, "AccountService.Create", "tenantID is required")
	}

	accType := domain.AccountTypeCash
	if input.AccountType != "" {
		accType = input.AccountType
	}

	currency := "COP"
	if input.Currency != "" {
		currency = input.Currency
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	isDefault := false
	if input.IsDefault != nil {
		isDefault = *input.IsDefault
	}

	// Validate rule: cannot set default if inactive
	if isDefault && !isActive {
		return nil, apperr.New(apperr.InvalidArgument, "AccountService.Create", "Cannot set an inactive account as default")
	}

	acc := &repo.Account{
		TenantID:    tenantID,
		Name:        input.Name,
		Code:        input.Code,
		AccountType: accType,
		Currency:    currency,
		IsActive:    isActive,
		IsDefault:   false, // We handle default separately to ensure transactionality
		Provider:    input.Provider,
		Notes:       input.Notes,
	}

	if err := s.repo.Create(ctx, acc); err != nil {
		// Detect unique violation for code
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperr.New(apperr.Conflict, "AccountService.Create", "An account with this code already exists")
		}
		return nil, apperr.Wrap(err, apperr.Internal, "AccountService.Create", "Failed to create account")
	}

	// If intended to be default, call set default tx
	if isDefault {
		if err := s.repo.SetDefaultTx(ctx, tenantID, acc.ID); err != nil {
			return nil, apperr.Wrap(err, apperr.Internal, "AccountService.Create", "Failed to set default account during creation")
		}
		acc.IsDefault = true
	}

	return acc, nil
}

func (s *accountService) GetAccountByID(ctx context.Context, tenantID, accountID string) (*repo.Account, error) {
	acc, err := s.repo.GetByID(ctx, tenantID, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.NotFound, "AccountService.Get", "Account not found")
		}
		return nil, apperr.Wrap(err, apperr.Internal, "AccountService.Get", "Failed to fetch account")
	}
	return acc, nil
}

func (s *accountService) GetDefaultAccount(ctx context.Context, tenantID string) (*repo.Account, error) {
	acc, err := s.repo.GetDefault(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.NotFound, "AccountService.GetDefault", "No default account found")
		}
		return nil, apperr.Wrap(err, apperr.Internal, "AccountService.GetDefault", "Failed to fetch default account")
	}
	return acc, nil
}

func (s *accountService) UpdateAccount(ctx context.Context, tenantID, accountID string, input UpdateAccountInput) (*repo.Account, error) {
	// First check if it exists & rules
	acc, err := s.GetAccountByID(ctx, tenantID, accountID)
	if err != nil {
		return nil, err
	}

	fields := repo.UpdateAccountFields{
		Name:        input.Name,
		Code:        input.Code,
		AccountType: input.AccountType,
		Currency:    input.Currency,
		IsActive:    input.IsActive,
		Provider:    input.Provider,
		Notes:       input.Notes,
	}

	// Rule: If deactivating an account that is the default, unset the default
	if input.IsActive != nil && !*input.IsActive && acc.IsDefault {
		// Update does not touch IsDefault, it needs to be un-set natively.
		// So we route this through ToggleStatus instead, to handle the unsetting properly.
		_, err := s.ToggleAccountStatus(ctx, tenantID, accountID, false)
		if err != nil {
			return nil, err
		}
		// Clear IsActive from fields so we don't update it twice
		fields.IsActive = nil
	}

	updated, err := s.repo.Update(ctx, tenantID, accountID, fields)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperr.New(apperr.Conflict, "AccountService.Update", "An account with this code already exists")
		}
		return nil, apperr.Wrap(err, apperr.Internal, "AccountService.Update", "Failed to update account")
	}

	return updated, nil
}

func (s *accountService) DeleteAccount(ctx context.Context, tenantID, accountID string) error {
	_, err := s.GetAccountByID(ctx, tenantID, accountID)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, tenantID, accountID)
	if err != nil {
		// Check for FK violation (transactions referencing this account)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			// Fallback: Soft delete by deactivating
			_, toggleErr := s.ToggleAccountStatus(ctx, tenantID, accountID, false)
			if toggleErr != nil {
				return apperr.Wrap(toggleErr, apperr.Internal, "AccountService.Delete", "Failed to soft-delete account after FK violation")
			}
			return nil // Return success as we successfully soft-deleted.
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "AccountService.Delete", "Account not found")
		}
		return apperr.Wrap(err, apperr.Internal, "AccountService.Delete", "Failed to delete account")
	}

	return nil
}

func (s *accountService) ListAccounts(ctx context.Context, tenantID string, filters repo.AccountFilters, page store.Page) (store.ResultList[repo.Account], error) {
	res, err := s.repo.List(ctx, tenantID, filters, page)
	if err != nil {
		return store.ResultList[repo.Account]{}, apperr.Wrap(err, apperr.Internal, "AccountService.List", "Failed to list accounts")
	}
	return res, nil
}

func (s *accountService) SetDefaultAccount(ctx context.Context, tenantID, accountID string) (*repo.Account, error) {
	acc, err := s.GetAccountByID(ctx, tenantID, accountID)
	if err != nil {
		return nil, err
	}

	if !acc.IsActive {
		return nil, apperr.New(apperr.InvalidArgument, "AccountService.SetDefault", "Cannot set an inactive account as default")
	}

	if err := s.repo.SetDefaultTx(ctx, tenantID, accountID); err != nil {
		return nil, apperr.Wrap(err, apperr.Internal, "AccountService.SetDefault", "Failed to set default account")
	}

	// Refetch correctly
	return s.GetAccountByID(ctx, tenantID, accountID)
}

func (s *accountService) ToggleAccountStatus(ctx context.Context, tenantID, accountID string, isActive bool) (*repo.Account, error) {
	_, err := s.GetAccountByID(ctx, tenantID, accountID)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.ToggleStatus(ctx, tenantID, accountID, isActive)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.NotFound, "AccountService.ToggleStatus", "Account not found")
		}
		return nil, apperr.Wrap(err, apperr.Internal, "AccountService.ToggleStatus", "Failed to toggle account status")
	}

	return updated, nil
}
