package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
)

type AuthenticatedUser struct {
	Identity auth.Identity
	User     api.User
}

type authenticatedUserKey struct{}

func WithAuthenticatedUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authenticatedUserKey{}, user)
}

func AuthenticatedUserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(authenticatedUserKey{}).(AuthenticatedUser)
	return user, ok
}
