package authcontext

import "context"

type actorIDKey struct{}

type adminKey struct{}

func WithActorID(ctx context.Context, actorID string) context.Context {
	return context.WithValue(ctx, actorIDKey{}, actorID)
}

func ActorID(ctx context.Context) (string, bool) {
	actorID, ok := ctx.Value(actorIDKey{}).(string)
	return actorID, ok && actorID != ""
}

// WithAdmin marks whether the authenticated actor holds a global
// administrator role. It is set once at the security boundary after bearer
// validation so handlers can gate privileged operations without extra
// upstream calls.
func WithAdmin(ctx context.Context, admin bool) context.Context {
	return context.WithValue(ctx, adminKey{}, admin)
}

func IsAdmin(ctx context.Context) bool {
	admin, _ := ctx.Value(adminKey{}).(bool)
	return admin
}
