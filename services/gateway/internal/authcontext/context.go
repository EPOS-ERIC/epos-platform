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

func ActorID(ctx context.Context) (string, bool) {
	user, ok := UserFromContext(ctx)
	return user.ID, ok && user.ID != ""
}
