package domain

import (
	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Location represents the public.locations table
type Location struct {
	db.TenantBaseModel

	Name    string  `gorm:"type:text;not null"`
	Code    *string `gorm:"type:text"`
	Phone   *string `gorm:"type:text"`
	Email   *string `gorm:"type:text"`
	Address *string `gorm:"type:text"`
	City    *string `gorm:"type:text"`
	Country *string `gorm:"type:text"`

	IsActive  bool `gorm:"not null;default:true"`
	IsDefault bool `gorm:"not null;default:false"`
}
