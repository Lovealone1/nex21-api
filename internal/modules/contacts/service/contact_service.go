package service

import (
	"context"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/contacts/repo"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"gorm.io/gorm"
)

var validContactTypes = map[string]bool{
	"customer": true,
	"supplier": true,
	"both":     true,
	"lead":     true,
}

var validLifecycleStages = map[string]bool{
	"lead":     true,
	"prospect": true,
	"customer": true,
	"inactive": true,
}

// ─── DTOs ────────────────────────────────────────────────────────────────────

type CreateContactInput struct {
	Name           string  `json:"name"`
	Email          *string `json:"email,omitempty"`
	Phone          *string `json:"phone,omitempty"`
	CompanyName    *string `json:"company_name,omitempty"`
	ContactType    string  `json:"contact_type,omitempty"`
	LifecycleStage string  `json:"lifecycle_stage,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
}

type UpdateContactInput struct {
	Name           *string `json:"name,omitempty"`
	Email          *string `json:"email,omitempty"`
	Phone          *string `json:"phone,omitempty"`
	CompanyName    *string `json:"company_name,omitempty"`
	ContactType    *string `json:"contact_type,omitempty"`
	LifecycleStage *string `json:"lifecycle_stage,omitempty"`
	Notes          *string `json:"notes,omitempty"`
}

// ─── Interface ───────────────────────────────────────────────────────────────

type ContactService interface {
	CreateContact(ctx context.Context, tenantID string, input CreateContactInput) (*repo.Contact, error)
	GetContactByID(ctx context.Context, tenantID, id string) (*repo.Contact, error)
	UpdateContact(ctx context.Context, tenantID, id string, input UpdateContactInput) (*repo.Contact, error)
	DeleteContact(ctx context.Context, tenantID, id string) error
	ListContacts(ctx context.Context, tenantID string, contactType string, page store.Page) (store.ResultList[repo.Contact], error)
	ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Contact, error)
	UpdateLifecycleStage(ctx context.Context, tenantID, id, stage string) (*repo.Contact, error)
	GetContactSummary(ctx context.Context, tenantID string) (*repo.ContactSummary, error)
}

// ─── Implementation ──────────────────────────────────────────────────────────

type contactService struct {
	repo repo.ContactRepo
}

func NewContactService(r repo.ContactRepo) ContactService {
	return &contactService{repo: r}
}

func (s *contactService) CreateContact(ctx context.Context, tenantID string, input CreateContactInput) (*repo.Contact, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Create", "tenantID is required")
	}
	if input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Create", "name is required")
	}

	if input.ContactType == "" {
		input.ContactType = "customer"
	} else if !validContactTypes[input.ContactType] {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Create", "invalid contact_type (must be customer, supplier, both, lead)")
	}

	if input.LifecycleStage == "" {
		input.LifecycleStage = "lead"
	} else if !validLifecycleStages[input.LifecycleStage] {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Create", "invalid lifecycle_stage")
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	c := &repo.Contact{
		TenantID:       tenantID,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		CompanyName:    input.CompanyName,
		ContactType:    input.ContactType,
		LifecycleStage: input.LifecycleStage,
		Notes:          input.Notes,
		IsActive:       isActive,
	}

	if err := s.repo.Create(ctx, c); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ContactService.Create", "Failed to create contact")
	}

	return c, nil
}

func (s *contactService) GetContactByID(ctx context.Context, tenantID, id string) (*repo.Contact, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.GetByID", "id and tenantID are required")
	}

	c, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ContactService.GetByID", "Contact not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ContactService.GetByID", "Failed to fetch contact")
	}

	return c, nil
}

func (s *contactService) UpdateContact(ctx context.Context, tenantID, id string, input UpdateContactInput) (*repo.Contact, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Update", "id and tenantID are required")
	}

	if input.Name != nil && *input.Name == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Update", "name cannot be empty")
	}

	if input.ContactType != nil && !validContactTypes[*input.ContactType] {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Update", "invalid contact_type")
	}

	if input.LifecycleStage != nil && !validLifecycleStages[*input.LifecycleStage] {
		return nil, errors.New(errors.InvalidArgument, "ContactService.Update", "invalid lifecycle_stage")
	}

	c, err := s.repo.Update(ctx, tenantID, id, repo.UpdateFields{
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		CompanyName:    input.CompanyName,
		ContactType:    input.ContactType,
		LifecycleStage: input.LifecycleStage,
		Notes:          input.Notes,
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ContactService.Update", "Contact not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ContactService.Update", "Failed to update contact")
	}

	return c, nil
}

func (s *contactService) DeleteContact(ctx context.Context, tenantID, id string) error {
	if id == "" || tenantID == "" {
		return errors.New(errors.InvalidArgument, "ContactService.Delete", "id and tenantID are required")
	}

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.NotFound, "ContactService.Delete", "Contact not found")
		}
		return errors.Wrap(err, errors.Internal, "ContactService.Delete", "Failed to delete contact")
	}

	return nil
}

func (s *contactService) ListContacts(ctx context.Context, tenantID, contactType string, page store.Page) (store.ResultList[repo.Contact], error) {
	if tenantID == "" {
		return store.ResultList[repo.Contact]{}, errors.New(errors.InvalidArgument, "ContactService.List", "tenantID is required")
	}

	if contactType != "" && !validContactTypes[contactType] {
		return store.ResultList[repo.Contact]{}, errors.New(errors.InvalidArgument, "ContactService.List", "invalid contactType filter")
	}

	result, err := s.repo.List(ctx, tenantID, contactType, page)
	if err != nil {
		return store.ResultList[repo.Contact]{}, errors.Wrap(err, errors.Internal, "ContactService.List", "Failed to list contacts")
	}

	return result, nil
}

func (s *contactService) ToggleStatus(ctx context.Context, tenantID, id string) (*repo.Contact, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.ToggleStatus", "id and tenantID are required")
	}

	c, err := s.repo.ToggleStatus(ctx, tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ContactService.ToggleStatus", "Contact not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ContactService.ToggleStatus", "Failed to toggle contact status")
	}

	return c, nil
}

func (s *contactService) UpdateLifecycleStage(ctx context.Context, tenantID, id, stage string) (*repo.Contact, error) {
	if id == "" || tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.UpdateLifecycleStage", "id and tenantID are required")
	}

	if !validLifecycleStages[stage] {
		return nil, errors.New(errors.InvalidArgument, "ContactService.UpdateLifecycleStage", "invalid lifecycle_stage")
	}

	c, err := s.repo.UpdateLifecycleStage(ctx, tenantID, id, stage)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "ContactService.UpdateLifecycleStage", "Contact not found")
		}
		return nil, errors.Wrap(err, errors.Internal, "ContactService.UpdateLifecycleStage", "Failed to update lifecycle stage")
	}

	return c, nil
}

func (s *contactService) GetContactSummary(ctx context.Context, tenantID string) (*repo.ContactSummary, error) {
	if tenantID == "" {
		return nil, errors.New(errors.InvalidArgument, "ContactService.GetContactSummary", "tenantID is required")
	}

	summary, err := s.repo.GetSummary(ctx, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "ContactService.GetContactSummary", "Failed to get contact summary")
	}

	return summary, nil
}
