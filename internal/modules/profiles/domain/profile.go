package domain

import (
	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// Profile is the GORM model that maps to the public.profiles table.
//
// Design note: unlike other modules, the primary key (ID) is NOT auto-generated
// by the database — it is intentionally set to the UUID issued by Supabase Auth
// (auth.users.id). GORM's autoIncrement and gen_random_uuid() defaults must be
// skipped; callers MUST provide the ID before calling Create.
//
// The module deliberately mixes two persistence strategies:
//   - Tenant-scoped operations (Create, GetByID) go through the TenantStore /
//     RLS layer using raw database/sql to guarantee isolation.
//   - Admin operations (AdminUpdate, AdminToggleStatus, AdminListAll) bypass RLS
//     using *sql.DB directly, since admin routes have no tenant context.
//
// This model exists purely as a schema reference and for any future tooling
// (e.g. migrations via GORM AutoMigrate) that needs a typed representation.
type Profile struct {
	// ID is the Supabase Auth UID — set externally, never auto-generated.
	ID string `gorm:"type:uuid;primaryKey"`

	// TenantID links the profile to a tenant. Nullable initially: the DB trigger
	// `on_auth_user_created` sets it from user_metadata right after insertion.
	TenantID *string `gorm:"type:uuid;index"`

	Email    *string `gorm:"type:text"`
	FullName *string `gorm:"type:text"`

	// Role must be one of: owner, admin, staff, member.
	Role string `gorm:"type:text;not null;default:member"`

	// IsActive controls whether the profile can access the system.
	IsActive bool `gorm:"type:boolean;not null;default:true"`

	db.BaseModel // CreatedAt, UpdatedAt (no TenantBaseModel: we supply TenantID manually)
}

// TableName tells GORM which table to use.
func (Profile) TableName() string {
	return "profiles"
}
