package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// PurchaseOrder represents the public.purchase_orders table
type PurchaseOrder struct {
	db.TenantBaseModel

	LocationID        string  `gorm:"type:uuid;not null"`
	SupplierContactID string  `gorm:"type:uuid;not null"`
	StaffID           *string `gorm:"type:uuid"`

	OrderNumber string `gorm:"type:text;not null;index:uq_tenant_po_number,unique"`
	Status      string `gorm:"type:text;not null;default:'draft'"`

	// Money Totals
	Subtotal      float64 `gorm:"type:numeric(12,2);not null;default:0"`
	DiscountTotal float64 `gorm:"type:numeric(12,2);not null;default:0"`
	TaxTotal      float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Total         float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency      string  `gorm:"type:text;not null;default:'COP'"`

	// Dates
	OrderedAt  *time.Time `gorm:"type:timestamptz"`
	ReceivedAt *time.Time `gorm:"type:timestamptz"`

	// Payment Tracking
	PaymentStatus string     `gorm:"type:text;not null;default:'unpaid'"`
	PaidTotal     float64    `gorm:"type:numeric(12,2);not null;default:0"`
	BalanceDue    float64    `gorm:"type:numeric(12,2);not null;default:0"`
	DueDate       *time.Time `gorm:"type:date"`

	Notes       *string `gorm:"type:text"`
	ExternalRef *string `gorm:"type:text"`

	CreatedBy *string `gorm:"type:uuid"`

	Lines []PurchaseOrderLine `gorm:"foreignKey:PurchaseOrderID"`
}

// PurchaseOrderLine represents the public.purchase_order_lines table
type PurchaseOrderLine struct {
	db.TenantBaseModel

	PurchaseOrderID string `gorm:"type:uuid;not null"`
	CatalogItemID   string `gorm:"type:uuid;not null"`

	Quantity  float64 `gorm:"type:numeric(12,2);not null;default:1"`
	UnitCost  float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Discount  float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Tax       float64 `gorm:"type:numeric(12,2);not null;default:0"`
	LineTotal float64 `gorm:"type:numeric(12,2);not null;default:0"`

	Notes *string `gorm:"type:text"`
}
