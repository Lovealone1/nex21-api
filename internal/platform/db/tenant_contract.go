package db

import (
	"context"
)

type contextKey string

const actorKey contextKey = "app_actor"

// Actor represents the unified identity for the current request context
type Actor struct {
	UserID   string
	TenantID string
	Role     string // e.g., "owner", "admin", "member"
}

// WithActor injects the Actor into the context.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

// ActorFrom safely extracts the Actor from the context.
func ActorFrom(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorKey).(Actor)
	return actor, ok
}

// MustActor extracts the Actor or panics. Use this in Handlers that are strictly behind the TenantMiddleware.
func MustActor(ctx context.Context) Actor {
	actor, ok := ActorFrom(ctx)
	if !ok {
		// Este panic será capturado por middleware.Recoverer y logueado como Error 500.
		// Previene que el código asuma datos si el middleware falló o fue omitido.
		panic("security block: strictly expected Actor in context but none found")
	}
	return actor
}

// MustTenant is a shorthand to get just the TenantID safely.
func MustTenant(ctx context.Context) string {
	return MustActor(ctx).TenantID
}
