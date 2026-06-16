package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/usergroup"
)

func authMiddleware(authenticator *auth.Service, provisioner *usergroup.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerTokenFromRequest(r)
			if !ok {
				handlers.WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
				return
			}

			identity, err := authenticator.Authenticate(r.Context(), token)
			if err != nil {
				if errors.Is(err, auth.ErrUnauthorized) {
					handlers.WriteError(r.Context(), w, http.StatusUnauthorized, "invalid_token", "token is invalid", nil)
					return
				}

				handlers.WriteError(r.Context(), w, http.StatusServiceUnavailable, "auth_service_unavailable", "authentication service unavailable", err)
				return
			}

			user, err := provisioner.EnsureUser(r.Context(), usergroup.Identity{
				AuthIdentifier: identity.AuthIdentifier,
				Email:          identity.Email,
				GivenName:      identity.GivenName,
				FamilyName:     identity.FamilyName,
			})
			if err != nil {
				handlers.WriteError(r.Context(), w, http.StatusServiceUnavailable, "user_provisioning_failed", "user provisioning failed", err)
				return
			}

			ctx := handlers.WithAuthenticatedUser(r.Context(), handlers.AuthenticatedUser{
				Identity: identity,
				User:     user,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerTokenFromRequest(r *http.Request) (string, bool) {
	head := strings.TrimSpace(r.Header.Get("Authorization"))
	if head == "" {
		return "", false
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(head, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(head, prefix))
	if token == "" {
		return "", false
	}

	return token, true
}
