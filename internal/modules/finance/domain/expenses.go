package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// ExpenseCategory represents the public.expense_categories table
type ExpenseCategory struct {
	db.TenantBaseModel

	Name        string  `gorm:"type:text;not null"`
	Code        *string `gorm:"type:text"`
	Description *string `gorm:"type:text"`

	CategoryType string `gorm:"type:text;not null;default:'opex'"` // 'opex', 'cogs', 'other'
	IsActive     bool   `gorm:"not null;default:true"`
}

// Expense represents the public.expenses table
type Expense struct {
	db.TenantBaseModel

	LocationID        *string `gorm:"type:uuid"`
	CategoryID        string  `gorm:"type:uuid;not null"`
	SupplierContactID *string `gorm:"type:uuid"`

	Reference   *string `gorm:"type:text"`
	Description string  `gorm:"type:text;not null"`

	ExpenseDate time.Time  `gorm:"type:date;not null"`
	DueDate     *time.Time `gorm:"type:date"`

	Status string `gorm:"type:text;not null;default:'draft'"` // 'draft', 'approved', 'paid', 'cancelled'

	Total    float64 `gorm:"type:numeric(12,2);not null"`
	Currency string  `gorm:"type:text;not null;default:'COP'"`

	CreatedBy *string `gorm:"type:uuid"`
}
