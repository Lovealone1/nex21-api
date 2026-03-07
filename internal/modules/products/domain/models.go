package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Product represents the public.products table.
// Each product belongs to a single tenant and tracks physical inventory.
type Product struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:products_tenant_id_idx"`

	// Identity fields
	Name        string  `gorm:"type:text;not null;index:products_tenant_name_idx"`
	Description *string `gorm:"type:text"`
	SKU         *string `gorm:"type:text;index:products_tenant_sku_idx"`

	// Pricing & Inventory
	Price    float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency string  `gorm:"type:text;not null;default:'COP'"`
	Quantity int     `gorm:"type:integer;not null;default:0"`

	// Status
	IsActive bool `gorm:"type:boolean;not null;default:true"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (Product) TableName() string {
	return "products"
}
