package server

import (
	"errors"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
)

func authMiddleware(authenticator *auth.Service, provisioner *clients.MetadataLifecycle) func(http.Handler) http.Handler {
	return authMiddlewareRequired(authenticator, provisioner)
}

func authMiddlewareRequired(authenticator *auth.Service, provisioner *clients.MetadataLifecycle) func(http.Handler) http.Handler {
	return authMiddlewareWithMode(authenticator, provisioner, true)
}

func authMiddlewareOptional(authenticator *auth.Service, provisioner *clients.MetadataLifecycle) func(http.Handler) http.Handler {
	return authMiddlewareWithMode(authenticator, provisioner, false)
}

func authMiddlewareWithMode(authenticator *auth.Service, provisioner *clients.MetadataLifecycle, required bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())

			token, ok := bearerTokenFromRequest(r)
			if !ok {
				if !required {
					next.ServeHTTP(w, r)
					return
				}

				span.SetAttributes(attribute.String("auth.result", "missing_token"))
				log.WarnContext(r.Context(), "missing bearer token")
				handlers.WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
				return
			}

			identity, err := authenticator.Authenticate(r.Context(), token)
			if err != nil {
				if errors.Is(err, auth.ErrUnauthorized) {
					span.SetAttributes(attribute.String("auth.result", "invalid_token"))
					log.WarnContext(r.Context(), "invalid bearer token")
					handlers.WriteError(r.Context(), w, http.StatusUnauthorized, "invalid_token", "token is invalid", nil)
					return
				}

				span.SetAttributes(attribute.String("auth.result", "auth_service_unavailable"))
				log.ErrorContext(r.Context(), "authentication service unavailable", "error", err)
				handlers.WriteError(r.Context(), w, http.StatusServiceUnavailable, "auth_service_unavailable", "authentication service unavailable", err)
				return
			}

			resolved, err := provisioner.EnsureUser(r.Context(), clients.ActorIdentity{
				AuthIdentifier: identity.AuthIdentifier,
				Email:          identity.Email,
				GivenName:      identity.GivenName,
				FamilyName:     identity.FamilyName,
			})
			if err != nil {
				span.SetAttributes(attribute.String("auth.result", "user_provisioning_failed"))
				log.ErrorContext(r.Context(), "user provisioning failed", "error", err)
				handlers.WriteError(r.Context(), w, http.StatusServiceUnavailable, "user_provisioning_failed", "user provisioning failed", err)
				return
			}

			user := handlers.ToAPIUser(resolved)
			ctx := logging.WithActor(r.Context(), user.ID)

			span.SetAttributes(
				attribute.String("auth.result", "authenticated"),
				// "enduser" is an otel-specific naming
				attribute.String("enduser.id", user.ID),
				attribute.Bool("enduser.is_global_admin", user.IsGlobalAdmin()),
			)

			ctx = handlers.WithAuthenticatedUser(ctx, user)

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
