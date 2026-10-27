package authcontext

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type userKey struct{}

func WithUser(ctx context.Context, user domain.User) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

func UserFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userKey{}).(domain.User)
	return user, ok
}

// ActorID returns the actorID string from a context, if not present it will return an empty string.
//
// TODO: why do we need this as a string? can we use a domain type for actor too? then why distinguish user & actor?
func ActorID(ctx context.Context) (string, bool) {
	user, ok := UserFromContext(ctx)
	return user.ID, ok && user.ID != ""
}
