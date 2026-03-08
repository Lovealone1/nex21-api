package staffdomain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// StaffRole represents the possible roles a staff member can have.
type StaffRole string

const (
	RoleOwner        StaffRole = "owner"
	RoleAdmin        StaffRole = "admin"
	RoleManager      StaffRole = "manager"
	RoleStaff        StaffRole = "staff"
	RoleReceptionist StaffRole = "receptionist"
)

// Staff represents the public.staff table.
type Staff struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:staff_tenant_id_idx"`

	// Link to other entities
	LocationID *string `gorm:"type:uuid;index:staff_tenant_location_idx"`
	ProfileID  *string `gorm:"type:uuid;index:staff_tenant_profile_idx"`

	// Identity fields
	DisplayName string  `gorm:"type:text;not null;index:staff_tenant_name_idx"`
	Email       *string `gorm:"type:text"`
	Phone       *string `gorm:"type:text"`

	// role & Status
	StaffRole StaffRole `gorm:"type:staff_role_type;not null;default:staff"`
	IsActive  bool      `gorm:"type:boolean;not null;default:true"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (Staff) TableName() string {
	return "staff"
}
