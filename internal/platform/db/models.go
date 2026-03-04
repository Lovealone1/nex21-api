package db

import (
	"time"
)

// BaseModel contains common columns for all tables.
type BaseModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// TenantBaseModel contains common columns for all tenant-scoped tables.
type TenantBaseModel struct {
	BaseModel
	TenantID string `gorm:"type:uuid;not null;index"`
}
