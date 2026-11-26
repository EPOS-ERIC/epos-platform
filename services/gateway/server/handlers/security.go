package handlers

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
	"go.opentelemetry.io/otel/attribute"
)

type securityHandler struct {
	auth     *auth.Service
	identity *identity.Service
}

func NewSecurityHandler(authService *auth.Service, identity *identity.Service) gatewayapi.SecurityHandler {
	return &securityHandler{
		auth:     authService,
		identity: identity,
	}
}

var _ gatewayapi.SecurityHandler = (*securityHandler)(nil)

func (s *securityHandler) HandleBearerAuth(ctx context.Context, operation gatewayapi.OperationName, token gatewayapi.BearerAuth) (context.Context, error) {
	// check if the provided token is a pat and if so handle it as such
	if ok := strings.HasPrefix(token.Token, "epos_pat_"); ok {
		digest := sha256.Sum256([]byte(token.Token))

		user, err := s.identity.UserForAPIToken(ctx, digest[:])
		if err != nil {
			if apperror.IsCode(err, apperror.CodeUnauthenticated) || apperror.IsCode(err, apperror.CodeNotFound) {
				return nil, apperror.Default(apperror.CodeUnauthenticated)
			}

			return nil, fmt.Errorf("authenticate API token: %w", securityFailure(err))
		}

		if user.Status == domain.UserStatusBlocked {
			return nil, apperror.Default(apperror.CodeUnauthenticated)
		}

		ctx = authenticatedContext(ctx, user, "api_token")

		// always reject token related operation for security reasons
		switch operation {
		case gatewayapi.IssueUserAPITokenOperation,
			gatewayapi.RotateUserAPITokenOperation,
			gatewayapi.RevokeUserAPITokenOperation,
			gatewayapi.RevokeSelfUserAPITokenOperation:
			return nil, apperror.Default(apperror.CodePermissionDenied)
		}

		return ctx, nil
	}

	// handle normal bearer oauth token

	identity, err := s.auth.Authenticate(ctx, token.Token)
	if err != nil {
		if !apperror.IsCode(err, apperror.CodeUnauthenticated) {
			return nil, fmt.Errorf("authenticate bearer token: %w", securityFailure(err))
		}

		return nil, fmt.Errorf("authenticate bearer token: %w", err)
	}

	observability.SetActorID(ctx, identity.AuthIdentifier)

	user, err := s.identity.EnsureUser(ctx, domain.ActorIdentity{
		AuthIdentifier: identity.AuthIdentifier,
		Email:          identity.Email,
		GivenName:      identity.GivenName,
		FamilyName:     identity.FamilyName,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure user: %w", securityFailure(err))
	}

	if user.Status == domain.UserStatusBlocked {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}

	return authenticatedContext(ctx, user, "oauth"), nil
}

// authenticatedContext records the authenticated user and credential type in request context and telemetry.
func authenticatedContext(ctx context.Context, user domain.User, credentialType string) context.Context {
	credential := attribute.String("auth.credential_type", credentialType)

	observability.SetActorID(ctx, user.ID)

	if labeler, ok := gatewayapi.LabelerFromContext(ctx); ok {
		labeler.Add(credential)
	}

	return authcontext.WithUser(ctx, user)
}

func securityFailure(err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}

	if errors.Is(err, context.DeadlineExceeded) || apperror.IsCode(err, apperror.CodeUpstreamTimeout) {
		return apperror.Wrap(apperror.CodeUpstreamTimeout, "", err)
	}

	return apperror.Wrap(apperror.CodeUpstreamUnavailable, "", err)
}
