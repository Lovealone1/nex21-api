package domain

import (
	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Contact represents the public.contacts table (CRM items)
type Contact struct {
	db.TenantBaseModel

	Name  string  `gorm:"type:text;not null"`
	Email *string `gorm:"type:text;index:contacts_email_idx"`
	Phone *string `gorm:"type:text;index:contacts_phone_idx"`

	CompanyName    *string `gorm:"type:text"`
	ContactType    string  `gorm:"type:text;not null;default:'customer'"`
	LifecycleStage string  `gorm:"type:text;default:'lead'"`
	Notes          *string `gorm:"type:text"`
	IsActive       bool    `gorm:"not null;default:true"`
}
