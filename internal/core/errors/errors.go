package errors

import "errors"

// Base sentinels that the TenantSession or Repository
// must return wrapping or absorbing the ones from Database.
var (
	ErrNotFound      = errors.New("resource not found")
	ErrConflict      = errors.New("resource conflict (already exists, unique violation, or state invalid)")
	ErrValidation    = errors.New("data validation failed")
	ErrInternal      = errors.New("internal storage engine error")
	ErrTenantMissing = errors.New("tenant context missing or actor unauthorized")
)
