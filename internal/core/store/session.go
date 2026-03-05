package store

import "context"

// UnitOfWork typed signature to encapsulate queries within a Tenant's Tx limit.
type UnitOfWork func(ctx context.Context, session TenantSession) error

// TenantSession acts as a persisted connection but caged to the Tenant Scope.
type TenantSession interface {
	Querier() Querier
	RunInTx(ctx context.Context, fn UnitOfWork) error

	// TenantID returns the original context pre-injected in the session
	TenantID() string
}

// GlobalSession acts as a free connection to the generic DB for shared resources.
type GlobalSession interface {
	Querier() Querier
	// ... global methods
}

// TenantStore is the main factory injected into Tenant-Scoped modules.
type TenantStore interface {
	// SessionFromContext refuses to start or resolve without a live "Actor" and "TenantID".
	SessionFromContext(ctx context.Context) (TenantSession, error)
}

// GlobalStore is strictly applied for dictionaries and generic shared logic.
type GlobalStore interface {
	Session(ctx context.Context) (GlobalSession, error)
}
