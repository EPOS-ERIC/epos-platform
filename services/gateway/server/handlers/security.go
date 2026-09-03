package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

type securityHandler struct {
	auth        *auth.Service
	provisioner *clients.MetadataLifecycle
}

func NewSecurityHandler(authService *auth.Service, provisioner *clients.MetadataLifecycle) gatewayapi.SecurityHandler {
	return &securityHandler{auth: authService, provisioner: provisioner}
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
	user, err := s.provisioner.EnsureUser(ctx, clients.ActorIdentity{
		AuthIdentifier: identity.AuthIdentifier,
		Email:          identity.Email,
		GivenName:      identity.GivenName,
		FamilyName:     identity.FamilyName,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errProvisioningUnavailable, err)
	}
	return withActor(ctx, user), nil
}
