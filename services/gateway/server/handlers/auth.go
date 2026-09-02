package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type authenticatedUserKey struct{}

func WithAuthenticatedUser(ctx context.Context, user domain.User) context.Context {
	return context.WithValue(ctx, authenticatedUserKey{}, user)
}

func AuthenticatedUserFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(authenticatedUserKey{}).(domain.User)
	return user, ok
}
