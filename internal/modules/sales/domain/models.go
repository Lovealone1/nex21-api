package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// SalesOrder represents the public.sales_orders table
type SalesOrder struct {
	db.TenantBaseModel

	LocationID string  `gorm:"type:uuid;not null"`
	ContactID  *string `gorm:"type:uuid"`
	StaffID    *string `gorm:"type:uuid"`

	OrderNumber   string     `gorm:"type:text;not null;index:uq_tenant_order_number,unique"`
	Status        string     `gorm:"type:text;not null;default:'draft'"`
	PaymentStatus string     `gorm:"type:text;not null;default:'unpaid'"`
	PaidAt        *time.Time `gorm:"type:timestamptz"`

	Subtotal      float64 `gorm:"type:numeric(12,2);not null;default:0"`
	DiscountTotal float64 `gorm:"type:numeric(12,2);not null;default:0"`
	TaxTotal      float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Total         float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency      string  `gorm:"type:text;not null;default:'COP'"`

	Notes       *string `gorm:"type:text"`
	ExternalRef *string `gorm:"type:text"`
	CreatedBy   *string `gorm:"type:uuid"`

	// Relationships
	Lines []SalesOrderLine `gorm:"foreignKey:SalesOrderID"`
}

// SalesOrderLine represents the public.sales_order_lines table
type SalesOrderLine struct {
	db.TenantBaseModel

	SalesOrderID  string `gorm:"type:uuid;not null"`
	CatalogItemID string `gorm:"type:uuid;not null"`

	Quantity  float64 `gorm:"type:numeric(12,2);not null;default:1"`
	UnitPrice float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Discount  float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Tax       float64 `gorm:"type:numeric(12,2);not null;default:0"`
	LineTotal float64 `gorm:"type:numeric(12,2);not null;default:0"`

	Notes *string `gorm:"type:text"`
}
