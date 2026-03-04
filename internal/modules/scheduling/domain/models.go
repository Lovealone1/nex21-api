package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Staff represents the public.staff table
type Staff struct {
	db.TenantBaseModel

	LocationID *string `gorm:"type:uuid"`
	ProfileID  *string `gorm:"type:uuid"`

	DisplayName string  `gorm:"type:text;not null"`
	Email       *string `gorm:"type:text"`
	Phone       *string `gorm:"type:text"`

	StaffRole string `gorm:"type:text;not null;default:'staff'"` // 'owner', 'admin', 'staff'
	IsActive  bool   `gorm:"not null;default:true"`
}

// WorkSchedule represents the public.work_schedules table
type WorkSchedule struct {
	db.TenantBaseModel

	StaffID    string  `gorm:"type:uuid;not null"`
	LocationID *string `gorm:"type:uuid"`

	Weekday   int16  `gorm:"type:smallint;not null"` // 0 = Sunday
	StartTime string `gorm:"type:time;not null"`     // e.g "09:00:00"
	EndTime   string `gorm:"type:time;not null"`

	IsActive bool `gorm:"not null;default:true"`
}

// Service represents the public.services table
type Service struct {
	db.TenantBaseModel

	LocationID *string `gorm:"type:uuid"`

	Name        string  `gorm:"type:text;not null"`
	Description *string `gorm:"type:text"`
	Category    *string `gorm:"type:text"`

	DurationMinutes int `gorm:"type:int;not null;default:30"`
	BufferMinutes   int `gorm:"type:int;not null;default:0"`

	Price    float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency string  `gorm:"type:text;not null;default:'COP'"`

	IsActive bool `gorm:"not null;default:true"`
}

// Appointment represents the public.appointments table
type Appointment struct {
	db.TenantBaseModel

	LocationID string `gorm:"type:uuid;not null"`
	StaffID    string `gorm:"type:uuid;not null"`
	ContactID  string `gorm:"type:uuid;not null"`
	ServiceID  string `gorm:"type:uuid;not null"`

	StartAt time.Time `gorm:"type:timestamptz;not null"`
	EndAt   time.Time `gorm:"type:timestamptz;not null"`

	Status string `gorm:"type:text;not null;default:'scheduled'"`

	Notes       *string    `gorm:"type:text"`
	CancelledAt *time.Time `gorm:"type:timestamptz"`
	CompletedAt *time.Time `gorm:"type:timestamptz"`
	CreatedBy   *string    `gorm:"type:uuid"`
}
