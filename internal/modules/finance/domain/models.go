package domain

import (
	"encoding/json"
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Transaction represents the public.transactions table
type Transaction struct {
	db.TenantBaseModel

	// Polymorphic targets
	AppointmentID   *string `gorm:"type:uuid"`
	SalesOrderID    *string `gorm:"type:uuid"`
	PurchaseOrderID *string `gorm:"type:uuid"`
	ExpenseID       *string `gorm:"type:uuid"`
	PayrollRunID    *string `gorm:"type:uuid"`

	// Account
	AccountID string `gorm:"type:uuid;not null"`

	// Party
	ContactID *string `gorm:"type:uuid"`

	// App Metadata
	LocationID *string `gorm:"type:uuid"`
	StaffID    *string `gorm:"type:uuid"`
	CreatedBy  *string `gorm:"type:uuid"`

	// Financial Core
	Direction string  `gorm:"type:text;not null"` // 'in' or 'out'
	Amount    float64 `gorm:"type:numeric(12,2);not null"`
	Currency  string  `gorm:"type:text;not null;default:'COP'"`
	Method    string  `gorm:"type:text;not null;default:'cash'"`
	Status    string  `gorm:"type:text;not null;default:'completed'"`

	PaidAt    *time.Time       `gorm:"type:timestamptz"`
	Reference *string          `gorm:"type:text"`
	Notes     *string          `gorm:"type:text"`
	Metadata  *json.RawMessage `gorm:"type:jsonb"`
}
