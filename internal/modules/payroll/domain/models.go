package payrolldomain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

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

// TableName overrides the table name used by GORM
func (StaffCompensation) TableName() string {
	return "staff_compensation"
}
