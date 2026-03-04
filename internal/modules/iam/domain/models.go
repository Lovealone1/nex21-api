package domain

import (
	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Profile represents the public.profiles table
// Note: This table is usually synchronized with auth.users via Supabase triggers
type Profile struct {
	db.BaseModel // Uses BaseModel since Profiles are global, NOT tenant-scoped

	Email     string  `gorm:"type:text;uniqueIndex;not null"`
	FirstName *string `gorm:"type:text"`
	LastName  *string `gorm:"type:text"`
	AvatarURL *string `gorm:"type:text"`
}
