package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

type authenticatedUserKey struct{}

func WithAuthenticatedUser(ctx context.Context, user api.User) context.Context {
	return context.WithValue(ctx, authenticatedUserKey{}, user)
}

func AuthenticatedUserFromContext(ctx context.Context) (api.User, bool) {
	user, ok := ctx.Value(authenticatedUserKey{}).(api.User)
	return user, ok
}
