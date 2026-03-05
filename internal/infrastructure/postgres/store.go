package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Lovealone1/nex21-api/internal/core/errors"
	"github.com/Lovealone1/nex21-api/internal/core/store"
	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// tenantStoreImpl is the real PostgreSQL implementation wrapping *sql.DB
type tenantStoreImpl struct {
	db *sql.DB
}

// NewTenantStore creates the base store injector
func NewTenantStore(db *sql.DB) store.TenantStore {
	return &tenantStoreImpl{
		db: db,
	}
}

func (s *tenantStoreImpl) SessionFromContext(ctx context.Context) (store.TenantSession, error) {
	// Re-using the existing Actor extraction from platform/db
	actor, ok := db.ActorFrom(ctx)
	if !ok || actor.TenantID == "" {
		return nil, errors.ErrTenantMissing
	}

	return &tenantSessionImpl{
		db:       s.db,
		tenantID: actor.TenantID,
	}, nil
}

type tenantSessionImpl struct {
	db       *sql.DB
	tenantID string
	tx       *sql.Tx // Active transaction if RunInTx was called
}

func (ts *tenantSessionImpl) TenantID() string {
	return ts.tenantID
}

func (ts *tenantSessionImpl) Querier() store.Querier {
	if ts.tx == nil {
		// Strict architectural decision: we force Tx to ensure RLS context applies securely.
		panic("Querier() called outside of RunInTx. RLS requires a transaction context.")
	}
	return &sqlQuerier{q: ts.tx}
}

// RunInTx opens the transaction, injects SET LOCAL for tenant context, and executes work.
func (ts *tenantSessionImpl) RunInTx(ctx context.Context, fn store.UnitOfWork) error {
	// If already in a Tx, just execute the function (nested logical transaction)
	if ts.tx != nil {
		return fn(ctx, ts)
	}

	// 1. Begin physical physical transaction
	tx, err := ts.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	// We defer a rollback. If the tx is already committed, rollback does nothing.
	defer tx.Rollback()

	// 2. RLS MAGIC: Inject context into postgres session natively via SET LOCAL.
	// We assume 'app.current_tenant' is what RLS policies check against.
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", ts.tenantID)
	if err != nil {
		return fmt.Errorf("failed to set RLS tenant context: %w", err)
	}

	// 3. Create a copy of session with the Tx reference injected
	txSession := &tenantSessionImpl{
		db:       ts.db,
		tenantID: ts.tenantID,
		tx:       tx,
	}

	// 4. Do the Repo work
	if err := fn(ctx, txSession); err != nil {
		return err // The defer will perform the rollback
	}

	// 5. Commit explicitly explicitly handling the result
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

// GlobalStore is independent from Tenant constraints if we ever need it.
type globalStoreImpl struct {
	db *sql.DB
}

func NewGlobalStore(db *sql.DB) store.GlobalStore {
	return &globalStoreImpl{db: db}
}

func (g *globalStoreImpl) Session(ctx context.Context) (store.GlobalSession, error) {
	return &globalSessionImpl{db: g.db}, nil
}

type globalSessionImpl struct {
	db *sql.DB
}

func (gs *globalSessionImpl) Querier() store.Querier {
	// A global session might just act as a wrapper around basic sql.DB querying without Tx limits
	return &sqlQuerier{q: gs.db}
}
