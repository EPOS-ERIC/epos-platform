package authcontext

import "context"

type actorIDKey struct{}

func WithActorID(ctx context.Context, actorID string) context.Context {
	return context.WithValue(ctx, actorIDKey{}, actorID)
}

func ActorID(ctx context.Context) (string, bool) {
	actorID, ok := ctx.Value(actorIDKey{}).(string)
	return actorID, ok && actorID != ""
}
