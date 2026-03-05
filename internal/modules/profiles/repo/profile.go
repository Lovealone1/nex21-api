package repo

import (
	"context"
	"time"

	"github.com/Lovealone1/nex21-api/internal/core/store"
)

// Profile represents the business entity for a user belonging to a specific Tenant.
// A Profile is always mapped 1:1 to a Supabase auth.users UID.
type Profile struct {
	ID        string    `json:"id"`        // This is the UUID from Supabase Auth
	TenantID  string    `json:"tenant_id"` // Tenant isolation boundary
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"` // e.g., 'admin', 'member'
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProfileRepo defines the contract for persisting Profiles in the current Tenant context.
type ProfileRepo interface {
	Create(ctx context.Context, profile *Profile) error
	GetByID(ctx context.Context, id string) (*Profile, error)
	// You might add List, Update, SoftDelete later.
}

type profileRepo struct {
	store store.TenantStore
}

// NewProfileRepo constructs a repository securely tied to the TenantStore abstraction.
func NewProfileRepo(s store.TenantStore) ProfileRepo {
	return &profileRepo{store: s}
}

func (r *profileRepo) Create(ctx context.Context, profile *Profile) error {
	session, err := r.store.SessionFromContext(ctx)
	if err != nil {
		return err
	}

	// We MUST execute all writes through RunInTx to guarantee RLS is applied.
	// If the TenantStore is Postgres, it injects SET LOCAL app.current_tenant = ?
	return session.RunInTx(ctx, func(ctx context.Context, txSession store.TenantSession) error {
		q := txSession.Querier()

		// Note: We don't need "WHERE tenant_id = ?" or explicitly inserting it
		// IF your RLS handles it and default column values use current_setting.
		// For safety and explicitness during INSERTs, we can still provide it.
		query := `
			INSERT INTO profiles (id, tenant_id, email, full_name, role)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING created_at, updated_at
		`

		err := q.QueryRow(ctx, query,
			profile.ID,
			txSession.TenantID(), // Getting tenant securely from session, not parameter
			profile.Email,
			profile.FullName,
			profile.Role,
		).Scan(&profile.CreatedAt, &profile.UpdatedAt)

		if err != nil {
			return err
		}

		profile.TenantID = txSession.TenantID()
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

		// The beauty of the Repo Contract: No "AND tenant_id = ?" needed.
		// PostgreSQL RLS handles the isolation via the Transaction's SET LOCAL.
		query := `
			SELECT id, tenant_id, email, full_name, role, created_at, updated_at
			FROM profiles
			WHERE id = $1
		`

		err := q.QueryRow(ctx, query, id).Scan(
			&p.ID,
			&p.TenantID,
			&p.Email,
			&p.FullName,
			&p.Role,
			&p.CreatedAt,
			&p.UpdatedAt,
		)

		return err
	})

	if err != nil {
		return nil, err // Errors are mapped generically (e.g. errors.ErrNotFound)
	}

	return &p, nil
}
