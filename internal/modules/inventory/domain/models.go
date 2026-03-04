package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// InventoryItem represents the public.inventory_items table
type InventoryItem struct {
	db.TenantBaseModel

	LocationID    string `gorm:"type:uuid;not null"`
	CatalogItemID string `gorm:"type:uuid;not null"`

	Quantity    float64 `gorm:"type:numeric(12,2);not null;default:0"`
	MinQuantity float64 `gorm:"type:numeric(12,2);not null;default:0"`

	UnitCost      *float64   `gorm:"type:numeric(12,2)"`
	LastCountedAt *time.Time `gorm:"type:timestamptz"`
}
