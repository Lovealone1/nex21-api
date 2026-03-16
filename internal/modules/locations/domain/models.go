package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Location represents the public.locations table.
type Location struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:locations_tenant_id_idx"`

	// Identity fields
	Name string  `gorm:"type:text;not null;index:locations_tenant_name_idx"`
	Code *string `gorm:"type:text;index:locations_tenant_code_idx"`

	// Optional contact fields
	Phone *string `gorm:"type:text"`
	Email *string `gorm:"type:text"`

	// Address
	Address *string `gorm:"type:text"`
	City    *string `gorm:"type:text"`
	Country *string `gorm:"type:text"`

	// Status
	IsActive  bool `gorm:"type:boolean;not null;default:true"`
	IsDefault bool `gorm:"type:boolean;not null;default:false"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (Location) TableName() string {
	return "locations"
}
