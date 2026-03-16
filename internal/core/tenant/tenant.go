package tenant

import "context"

// TenantID identifica inequívocamente la instancia / workspace base de clientes.
type TenantID string

// Actor representa el individuo/API que está ejecutando la acción actualmente.
type Actor struct {
	ID       string
	TenantID TenantID
	Roles    []string
}

// contextKey tipo oscuro para prevenir envenenamiento externo de keys a nivel string.
type contextKey string

const actorCtxKey contextKey = "ctx_tenant_actor"

// InjectActor inserta la unidad mínima en el contexto para viajar por la aplicación.
// Debería usarse de manera exclusiva por los Auth Middlewares o pipelines GRPC/HTTP.
func InjectActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorCtxKey, actor)
}

// ExtractActor recobra el estado de autorización/tenant de las peticiones en vuelo.
func ExtractActor(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorCtxKey).(Actor)
	return actor, ok
}
