package domain

import (
	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// CatalogItem represents the public.catalog_items table
type CatalogItem struct {
	db.TenantBaseModel

	ItemType    string  `gorm:"type:text;not null"` // 'product' or 'service'
	Name        string  `gorm:"type:text;not null"`
	Description *string `gorm:"type:text"`
	SKU         *string `gorm:"type:text"` // unique per tenant

	Price    float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency string  `gorm:"type:text;not null;default:'COP'"`
	IsActive bool    `gorm:"not null;default:true"`
}
