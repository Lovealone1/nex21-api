package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
)

// Profile represents the business entity for a user belonging to a specific Tenant.
// A Profile is always mapped 1:1 to a Supabase auth.users UID.
type Profile struct {
	ID        string    `json:"id"`        // This is the UUID from Supabase Auth
	TenantID  *string   `json:"tenant_id"` // Tenant isolation boundary (Nullable for unbound users)
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`      // System role
	IsActive  bool      `json:"is_active"` // Whether the profile is enabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateFields holds the optional fields that can be patched on a Profile.
// Use pointer types so callers can send only the fields they want to change.
type UpdateFields struct {
	FullName *string
	Email    *string
	Role     *string
	IsActive *bool
}

// ProfileRepo defines the contract for persisting Profiles.
type ProfileRepo interface {
	Create(ctx context.Context, profile *Profile) error
	GetByID(ctx context.Context, id string) (*Profile, error)
	// AdminUpdate patches a profile row directly (no RLS). For admin routes only.
	AdminUpdate(ctx context.Context, id string, fields UpdateFields) (*Profile, error)
	// AdminGetByID fetches a profile row directly (no RLS). For admin routes only.
	AdminGetByID(ctx context.Context, id string) (*Profile, error)
	// AdminToggleStatus atomically inverts the is_active flag. For admin routes only.
	AdminToggleStatus(ctx context.Context, id string) (*Profile, error)
	// AdminListAll returns a paginated list of all profiles (no RLS). For admin routes only.
	AdminListAll(ctx context.Context, page store.Page) (store.ResultList[Profile], error)
}

type profileRepo struct {
	store store.TenantStore
	db    *sql.DB // raw connection for admin (no-RLS) operations
}

// NewProfileRepo constructs a repository.
// db is the raw *sql.DB used for admin operations that bypass tenant RLS.
func NewProfileRepo(s store.TenantStore, db *sql.DB) ProfileRepo {
	return &profileRepo{store: s, db: db}
}

// ─── Tenant-scoped operations (require TenantMiddleware in the request) ──────

func (r *profileRepo) Create(ctx context.Context, profile *Profile) error {
	session, err := r.store.SessionFromContext(ctx)
	if err != nil {
		return err
	}

	return session.RunInTx(ctx, func(ctx context.Context, txSession store.TenantSession) error {
		q := txSession.Querier()

		query := `
			INSERT INTO profiles (id, tenant_id, email, full_name, role)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING created_at, updated_at
		`

		err := q.QueryRow(ctx, query,
			profile.ID,
			profile.TenantID, // can be nil
			profile.Email,
			profile.FullName,
			profile.Role,
		).Scan(&profile.CreatedAt, &profile.UpdatedAt)

		if err != nil {
			return err
		}

		return nil
	})
}

func (r *profileRepo) GetByID(ctx context.Context, id string) (*Profile, error) {
	session, err := r.store.SessionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var p Profile

	err = session.RunInTx(ctx, func(ctx context.Context, txSession store.TenantSession) error {
		q := txSession.Querier()

		query := `
			SELECT id, tenant_id, email, full_name, role, is_active, created_at, updated_at
			FROM profiles
			WHERE id = $1
		`

		var nullTenant sql.NullString
		err := q.QueryRow(ctx, query, id).Scan(
			&p.ID,
			&nullTenant,
			&p.Email,
			&p.FullName,
			&p.Role,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
		)

		if nullTenant.Valid {
			p.TenantID = &nullTenant.String
		}
		return err
	})

	if err != nil {
		return nil, err
	}

	return &p, nil
}

// ─── Admin operations (no RLS — admin routes only) ───────────────────────────

// AdminGetByID fetches a profile by id without going through the TenantStore.
// Used by admin services that have no tenant context in their request.
func (r *profileRepo) AdminGetByID(ctx context.Context, id string) (*Profile, error) {
	query := `
		SELECT id, tenant_id, email, full_name, role, is_active, created_at, updated_at
		FROM profiles
		WHERE id = $1
	`

	var p Profile
	var tenantID, email, fullName sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&tenantID,
		&email,
		&fullName,
		&p.Role,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	if tenantID.Valid {
		p.TenantID = &tenantID.String
	}
	p.Email = email.String
	p.FullName = fullName.String

	return &p, nil
}

// AdminUpdate patches a profile row without going through the TenantStore.
// Only non-nil fields in UpdateFields are written.
// Used by admin services that have no tenant context in their request.
func (r *profileRepo) AdminUpdate(ctx context.Context, id string, fields UpdateFields) (*Profile, error) {
	query := `
		UPDATE profiles
		SET
			full_name  = COALESCE($2, full_name),
			email      = COALESCE($3, email),
			role       = COALESCE($4, role),
			is_active  = COALESCE($5, is_active),
			updated_at = now()
		WHERE id = $1
		RETURNING id, tenant_id, email, full_name, role, is_active, created_at, updated_at
	`

	var p Profile
	var tenantID, email, fullName sql.NullString
	err := r.db.QueryRowContext(ctx, query,
		id,
		fields.FullName,
		fields.Email,
		fields.Role,
		fields.IsActive,
	).Scan(
		&p.ID,
		&tenantID,
		&email,
		&fullName,
		&p.Role,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	if tenantID.Valid {
		p.TenantID = &tenantID.String
	}
	p.Email = email.String
	p.FullName = fullName.String

	return &p, nil
}

// AdminToggleStatus atomically inverts the is_active flag without going through
// the TenantStore. Uses a single UPDATE ... SET is_active = NOT is_active to
// avoid the pgx prepared-statement conflict (42P05) that happens when two
// separate queries share the same pool connection in the same request.
func (r *profileRepo) AdminToggleStatus(ctx context.Context, id string) (*Profile, error) {
	query := `
		UPDATE profiles
		SET
			is_active  = NOT is_active,
			updated_at = now()
		WHERE id = $1
		RETURNING id, tenant_id, email, full_name, role, is_active, created_at, updated_at
	`

	var p Profile
	var tenantID, email, fullName sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&tenantID,
		&email,
		&fullName,
		&p.Role,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	if tenantID.Valid {
		p.TenantID = &tenantID.String
	}
	p.Email = email.String
	p.FullName = fullName.String

	return &p, nil
}

// sortableColumns is the allowlist of profile columns that can be used for ORDER BY.
// Never interpolate user input directly — always check against this map first.
var sortableColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"email":      true,
	"full_name":  true,
	"role":       true,
	"is_active":  true,
}

// AdminListAll returns a paginated list of all profiles without RLS.
// Uses COUNT(*) OVER() in a single query to avoid the pgx prepared-statement
// conflict (42P05) that occurs when a COUNT + SELECT pair shares a pool connection.
func (r *profileRepo) AdminListAll(ctx context.Context, page store.Page) (store.ResultList[Profile], error) {
	// Build ORDER BY from validated sorts (allowlist prevents SQL injection).
	orderBy := "created_at DESC"
	if len(page.Sorts) > 0 {
		s := page.Sorts[0]
		if sortableColumns[s.Field] {
			dir := "DESC"
			if s.Direction == store.SortAsc {
				dir = "ASC"
			}
			orderBy = fmt.Sprintf("%s %s", s.Field, dir)
		}
	}

	// Single query: window function returns total alongside every row.
	// No second round-trip, no prepared-statement collision.
	query := fmt.Sprintf(`
		SELECT id, tenant_id, email, full_name, role, is_active, created_at, updated_at,
		       COUNT(*) OVER() AS total_count
		FROM profiles
		ORDER BY %s
		LIMIT $1 OFFSET $2
	`, orderBy)

	rows, err := r.db.QueryContext(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return store.ResultList[Profile]{}, err
	}
	defer rows.Close()

	var profiles []Profile
	var total int64

	for rows.Next() {
		var p Profile
		var tenantID, email, fullName sql.NullString
		if err := rows.Scan(
			&p.ID,
			&tenantID,
			&email,
			&fullName,
			&p.Role,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
			&total, // COUNT(*) OVER()
		); err != nil {
			return store.ResultList[Profile]{}, err
		}
		if tenantID.Valid {
			p.TenantID = &tenantID.String
		}
		p.Email = email.String
		p.FullName = fullName.String
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return store.ResultList[Profile]{}, err
	}

	return store.ResultList[Profile]{
		Items: profiles,
		Total: total,
		Page:  page,
	}, nil
}
