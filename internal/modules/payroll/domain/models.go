// Updated Models for Payroll Runs and Items -> adding them to domain/models.go
package payrolldomain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// -----------------------------------------------------------------------------
// ENUMS
// -----------------------------------------------------------------------------

type RunFrequency string

const (
	FrequencyDaily    RunFrequency = "daily"
	FrequencyWeekly   RunFrequency = "weekly"
	FrequencyBiweekly RunFrequency = "biweekly"
	FrequencyMonthly  RunFrequency = "monthly"
	FrequencyCustom   RunFrequency = "custom"
)

type RunStatus string

const (
	StatusDraft     RunStatus = "draft"
	StatusApproved  RunStatus = "approved"
	StatusPaid      RunStatus = "paid"
	StatusCancelled RunStatus = "cancelled"
)

type LineType string

const (
	LineTypeEarning   LineType = "earning"
	LineTypeDeduction LineType = "deduction"
)

type ItemConcept string

const (
	ConceptSalary     ItemConcept = "salary"
	ConceptCommission ItemConcept = "commission"
	ConceptBonus      ItemConcept = "bonus"
	ConceptTips       ItemConcept = "tips"
	ConceptOvertime   ItemConcept = "overtime"
	ConceptAdvance    ItemConcept = "advance"
	ConceptLoan       ItemConcept = "loan"
	ConceptOther      ItemConcept = "other"
)

// -----------------------------------------------------------------------------
// MODELS
// -----------------------------------------------------------------------------

// StaffCompensation represents the public.staff_compensation table.
type StaffCompensation struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:staff_comp_tenant_staff_idx"`
	StaffID  string `gorm:"type:uuid;not null;index:staff_comp_tenant_staff_idx"`

	Scheme        string   `gorm:"type:text;not null;default:fixed"`
	PayFrequency  string   `gorm:"type:text;not null;default:monthly"`
	BaseSalary    *float64 `gorm:"type:numeric(12,2)"`
	HourlyRate    *float64 `gorm:"type:numeric(12,2)"`
	CommissionPct *float64 `gorm:"type:numeric(5,4)"`

	EffectiveFrom time.Time  `gorm:"type:date;not null"`
	EffectiveTo   *time.Time `gorm:"type:date"`
	IsActive      bool       `gorm:"type:boolean;not null;default:true;index:staff_comp_tenant_active_idx"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (StaffCompensation) TableName() string {
	return "staff_compensation"
}

// PayrollRun represents the public.payroll_runs table.
type PayrollRun struct {
	db.BaseModel
	TenantID   string  `gorm:"type:uuid;not null;index:payroll_runs_tenant_period_idx"`
	LocationID *string `gorm:"type:uuid"`

	Frequency   RunFrequency `gorm:"type:text;not null;default:monthly"`
	PeriodStart time.Time    `gorm:"type:date;not null;index:payroll_runs_tenant_period_idx"`
	PeriodEnd   time.Time    `gorm:"type:date;not null;index:payroll_runs_tenant_period_idx"`
	PayDate     time.Time    `gorm:"type:date;not null"`

	Status   RunStatus `gorm:"type:text;not null;default:draft;index:payroll_runs_tenant_status_idx"`
	Total    float64   `gorm:"type:numeric(12,2);not null;default:0"`
	Currency string    `gorm:"type:text;not null;default:COP"`
	Notes    *string   `gorm:"type:text"`

	CreatedBy *string   `gorm:"type:uuid"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (PayrollRun) TableName() string {
	return "payroll_runs"
}

// PayrollItem represents the public.payroll_items table.
type PayrollItem struct {
	db.BaseModel
	TenantID     string `gorm:"type:uuid;not null;index:payroll_items_tenant_staff_idx;index:payroll_items_tenant_concept_idx"`
	PayrollRunID string `gorm:"type:uuid;not null;index:payroll_items_run_idx"`
	StaffID      string `gorm:"type:uuid;not null;index:payroll_items_tenant_staff_idx"`

	LineType LineType    `gorm:"type:text;not null;default:earning"`
	Concept  ItemConcept `gorm:"type:text;not null;default:salary;index:payroll_items_tenant_concept_idx"`
	Amount   float64     `gorm:"type:numeric(12,2);not null"`

	AppointmentID *string `gorm:"type:uuid"`
	ServiceID     *string `gorm:"type:uuid"`
	Notes         *string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (PayrollItem) TableName() string {
	return "payroll_items"
}
