package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/modules/profiles/domain"
	"gorm.io/gorm"
)

// Member is the aggregate business entity returned by the repository layer.
type Member struct {
	ID       string `json:"id"`        // This is the membership ID
	TenantID string `json:"tenant_id"` // The tenant they belong to
	UserID   string `json:"user_id"`   // The user's ID (profile ID / auth UID)
	Role     string `json:"role"`      // Role in this specific tenant
	Status   string `json:"status"`    // Status in this specific tenant (active/inactive)

	// Enriched data from User Profile
	Email    string `json:"email"`
	FullName string `json:"full_name"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MembershipModel maps to public.memberships and specifies its profile relation
type MembershipModel struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID  string         `gorm:"type:uuid;not null"`
	UserID    string         `gorm:"type:uuid;not null"`
	Role      string         `gorm:"type:text;not null"`
	Status    string         `gorm:"type:text;not null"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	Profile   domain.Profile `gorm:"foreignKey:UserID;references:ID"`
}

// TableName matches the public.memberships table
func (MembershipModel) TableName() string {
	return "memberships"
}

// toAggregate maps from the DB model to the API entity
func (m *MembershipModel) toAggregate() *Member {
	email := ""
	if m.Profile.Email != nil {
		email = *m.Profile.Email
	}
	fullName := ""
	if m.Profile.FullName != nil {
		fullName = *m.Profile.FullName
	}

	return &Member{
		ID:        m.ID,
		TenantID:  m.TenantID,
		UserID:    m.UserID,
		Role:      m.Role,
		Status:    m.Status,
		Email:     email,
		FullName:  fullName,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// MemberRepo defines the persistence contract for Tenant Members.
type MemberRepo interface {
	// AddMember inserts a new membership row linking a user to a tenant.
	AddMember(ctx context.Context, tenantID, userID, role string) (*Member, error)
	// GetMember fetches a single member by tenantID and userID.
	GetMember(ctx context.Context, tenantID, userID string) (*Member, error)
	// UpdateRole changes a member's role within the tenant.
	UpdateRole(ctx context.Context, tenantID, userID, role string) (*Member, error)
	// ToggleStatus atomically flips a member's status between "active" and "inactive".
	ToggleStatus(ctx context.Context, tenantID, userID string) (*Member, error)
	// RemoveMember permanently deletes the membership link.
	RemoveMember(ctx context.Context, tenantID, userID string) error
	// ListMembers returns a paginated list of members for a specific tenant.
	ListMembers(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Member], error)
}

type memberRepo struct {
	db *gorm.DB
}

// NewMemberRepo creates a repository backed by the given gorm.DB instance.
func NewMemberRepo(db *gorm.DB) MemberRepo {
	return &memberRepo{db: db}
}

// ─── AddMember ────────────────────────────────────────────────────────────────

func (r *memberRepo) AddMember(ctx context.Context, tenantID, userID, role string) (*Member, error) {
	model := MembershipModel{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		Status:   "active",
	}

	result := r.db.WithContext(ctx).Create(&model)
	if result.Error != nil {
		return nil, result.Error
	}

	return r.GetMember(ctx, tenantID, userID)
}

// ─── GetMember ────────────────────────────────────────────────────────────────

func (r *memberRepo) GetMember(ctx context.Context, tenantID, userID string) (*Member, error) {
	var m MembershipModel
	result := r.db.WithContext(ctx).
		Preload("Profile").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&m)

	if result.Error != nil {
		return nil, result.Error
	}
	return m.toAggregate(), nil
}

// ─── UpdateRole ───────────────────────────────────────────────────────────────

func (r *memberRepo) UpdateRole(ctx context.Context, tenantID, userID, role string) (*Member, error) {
	result := r.db.WithContext(ctx).Model(&MembershipModel{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Update("role", role)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return r.GetMember(ctx, tenantID, userID)
}

// ─── ToggleStatus ──────────────────────────────────────────────────────────────

func (r *memberRepo) ToggleStatus(ctx context.Context, tenantID, userID string) (*Member, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE memberships
		SET status = CASE WHEN status = 'active' THEN 'inactive' ELSE 'active' END
		WHERE tenant_id = ? AND user_id = ?
	`, tenantID, userID)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// Touch the updated_at timestamp since we ran a raw query
	r.db.WithContext(ctx).Model(&MembershipModel{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Update("updated_at", time.Now())

	return r.GetMember(ctx, tenantID, userID)
}

// ─── RemoveMember ─────────────────────────────────────────────────────────────

func (r *memberRepo) RemoveMember(ctx context.Context, tenantID, userID string) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Delete(&MembershipModel{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ─── ListMembers ──────────────────────────────────────────────────────────────

// sortableMemberColumns prevents SQL injection in ORDER BY.
var sortableMemberColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"role":       true,
	"status":     true,
	"email":      true,
	"full_name":  true,
}

func (r *memberRepo) ListMembers(ctx context.Context, tenantID string, page store.Page) (store.ResultList[Member], error) {
	orderBy := "memberships.created_at DESC"
	if len(page.Sorts) > 0 {
		s := page.Sorts[0]
		field := s.Field
		if sortableMemberColumns[field] {
			// Prefix fields correctly depending on which table they come from
			if field == "email" || field == "full_name" {
				field = "profiles." + field
			} else {
				field = "memberships." + field
			}

			dir := "DESC"
			if s.Direction == store.SortAsc {
				dir = "ASC"
			}
			orderBy = fmt.Sprintf("%s %s", field, dir)
		}
	}

	var models []MembershipModel
	var total int64

	// Separated queries to dodge pgx 42P05 cache collision
	countResult := r.db.WithContext(ctx).Model(&MembershipModel{}).
		Where("memberships.tenant_id = ?", tenantID).
		Count(&total)

	if countResult.Error != nil {
		return store.ResultList[Member]{}, countResult.Error
	}

	result := r.db.WithContext(ctx).
		Preload("Profile").
		Joins("JOIN profiles ON profiles.id = memberships.user_id").
		Where("memberships.tenant_id = ?", tenantID).
		Order(orderBy).
		Offset(page.Offset).
		Limit(page.Limit).
		Find(&models)

	if result.Error != nil {
		return store.ResultList[Member]{}, result.Error
	}

	// Map to business aggregate
	members := make([]Member, 0, len(models))
	for _, raw := range models {
		members = append(members, *raw.toAggregate())
	}

	return store.ResultList[Member]{
		Items: members,
		Total: total,
		Page:  page,
	}, nil
}
