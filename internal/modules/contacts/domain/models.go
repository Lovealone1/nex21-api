package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Contact represents the public.contacts table
// It does NOT strictly belong to a "Tenant" domain aggregate since it will be globally
// managed via generic admin endpoints, though it is linked to one via TenantID.
type Contact struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:contacts_tenant_id_idx"`

	// Identity fields
	Name  string  `gorm:"type:text;not null;index:contacts_tenant_id_name_idx"`
	Email *string `gorm:"type:text;index:contacts_email_idx"`
	Phone *string `gorm:"type:text;index:contacts_phone_idx"`

	// Classification & CRM fields
	CompanyName    *string `gorm:"type:text"`
	ContactType    string  `gorm:"type:contact_type;not null;default:'customer'"`
	LifecycleStage string  `gorm:"type:text;default:'lead'"`
	Notes          *string `gorm:"type:text"`

	// Status
	IsActive bool `gorm:"type:boolean;not null;default:true"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (Contact) TableName() string {
	return "contacts"
}
