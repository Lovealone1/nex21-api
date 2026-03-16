package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Tenant represents the public.tenants table
type Tenant struct {
	db.BaseModel
	Name     string `gorm:"type:text;not null"`
	Slug     string `gorm:"type:text;not null;unique"`
	Plan     string `gorm:"type:text;not null;default:'free'"`
	IsActive bool   `gorm:"type:boolean;not null;default:true"`

	// Relationships
	Domains     []TenantDomain `gorm:"foreignKey:TenantID"`
	Memberships []Membership   `gorm:"foreignKey:TenantID"`
}

// TenantDomain represents the public.tenant_domains table
type TenantDomain struct {
	db.BaseModel
	TenantID   string     `gorm:"type:uuid;not null;index"`
	Domain     string     `gorm:"type:text;not null;unique"`
	DomainType string     `gorm:"type:text;not null;default:'subdomain'"`
	IsPrimary  bool       `gorm:"not null;default:true"`
	VerifiedAt *time.Time `gorm:"type:timestamptz"`
}

// Membership represents the public.memberships table
type Membership struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:memberships_tenant_id_idx"`
	UserID   string `gorm:"type:uuid;not null;index:memberships_user_id_idx"`
	Role     string `gorm:"type:text;not null;default:'member'"`
	IsActive bool   `gorm:"type:boolean;not null;default:true"`
}
