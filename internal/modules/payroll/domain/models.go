package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// StaffCompensation represents the public.staff_compensation table
type StaffCompensation struct {
	db.TenantBaseModel

	StaffID string `gorm:"type:uuid;not null"`

	Scheme       string `gorm:"type:text;not null;default:'fixed'"`
	PayFrequency string `gorm:"type:text;not null;default:'monthly'"`

	BaseSalary    *float64 `gorm:"type:numeric(12,2)"`
	HourlyRate    *float64 `gorm:"type:numeric(12,2)"`
	CommissionPct *float64 `gorm:"type:numeric(5,4)"`

	EffectiveFrom time.Time  `gorm:"type:date;not null"`
	EffectiveTo   *time.Time `gorm:"type:date"`

	IsActive bool `gorm:"not null;default:true"`
}

// PayrollRun represents the public.payroll_runs table
type PayrollRun struct {
	db.TenantBaseModel

	LocationID *string `gorm:"type:uuid"`

	Frequency   string    `gorm:"type:text;not null;default:'monthly'"`
	PeriodStart time.Time `gorm:"type:date;not null"`
	PeriodEnd   time.Time `gorm:"type:date;not null"`
	PayDate     time.Time `gorm:"type:date;not null"`

	Status   string  `gorm:"type:text;not null;default:'draft'"`
	Total    float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency string  `gorm:"type:text;not null;default:'COP'"`
	Notes    *string `gorm:"type:text"`

	CreatedBy *string `gorm:"type:uuid"`

	Items []PayrollItem `gorm:"foreignKey:PayrollRunID"`
}

// PayrollItem represents the public.payroll_items table
type PayrollItem struct {
	db.TenantBaseModel

	PayrollRunID string `gorm:"type:uuid;not null"`
	StaffID      string `gorm:"type:uuid;not null"`

	LineType string  `gorm:"type:text;not null;default:'earning'"` // 'earning', 'deduction'
	Concept  string  `gorm:"type:text;not null;default:'salary'"`
	Amount   float64 `gorm:"type:numeric(12,2);not null"`

	AppointmentID *string `gorm:"type:uuid"`
	ServiceID     *string `gorm:"type:uuid"`
	Notes         *string `gorm:"type:text"`
}
