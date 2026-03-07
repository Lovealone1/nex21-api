package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Service represents the public.services table.
// Each service belongs to a single tenant and describes a product/service
// the business offers to its customers.
type Service struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:services_tenant_id_idx"`

	// Identity fields
	Name        string  `gorm:"type:text;not null;index:services_tenant_id_name_idx"`
	Description *string `gorm:"type:text"`
	SKU         *string `gorm:"type:text;index:services_tenant_sku_idx"`

	// Scheduling defaults
	DurationMinutes int `gorm:"type:integer;not null;default:30"`
	BufferMinutes   int `gorm:"type:integer;not null;default:0"`

	// Pricing
	Price    float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency string  `gorm:"type:text;not null;default:'COP'"`

	// Classification
	Category *string `gorm:"type:text"`

	// Status
	IsActive bool `gorm:"type:boolean;not null;default:true"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (Service) TableName() string {
	return "services"
}
