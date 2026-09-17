package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
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

func (s *securityHandler) HandleBearerAuth(ctx context.Context, _ gatewayapi.OperationName, token gatewayapi.BearerAuth) (context.Context, error) {
	identity, err := s.auth.Authenticate(ctx, token.Token)
	if err != nil {
		if !errors.Is(err, auth.ErrUnauthorized) {
			return nil, fmt.Errorf("%w: %w", errAuthUnavailable, err)
		}
		return nil, fmt.Errorf("authenticate bearer token: %w", err)
	}

	ctx = authcontext.WithActorID(ctx, identity.AuthIdentifier)
	observability.SetActorID(ctx, identity.AuthIdentifier)

	user, err := s.identity.EnsureUser(ctx, domain.ActorIdentity{
		AuthIdentifier: identity.AuthIdentifier,
		Email:          identity.Email,
		GivenName:      identity.GivenName,
		FamilyName:     identity.FamilyName,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errProvisioningUnavailable, err)
	}
	if user.Status == domain.UserStatusBlocked {
		return nil, domain.ErrForbidden
	}

	return ctx, nil
}
