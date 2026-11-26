package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) IssueUserAPIToken(ctx context.Context) (*gatewayapi.UserAPITokenHeaders, error) {
	return h.issueUserAPIToken(ctx, false)
}

func (h *ogenHandler) RotateUserAPIToken(ctx context.Context) (*gatewayapi.UserAPITokenHeaders, error) {
	return h.issueUserAPIToken(ctx, true)
}

func (h *ogenHandler) issueUserAPIToken(ctx context.Context, rotate bool) (*gatewayapi.UserAPITokenHeaders, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	token, err := h.metadataLifecycleClient.IssueUserAPIToken(ctx, actor, rotate)
	if err != nil {
		return nil, fmt.Errorf("issue user API token: %w", err)
	}

	return &gatewayapi.UserAPITokenHeaders{
		CacheControl: gatewayapi.PrepareDistributionOKCacheControlNoStore,
		Response:     gatewayapi.UserAPIToken{Token: token},
	}, nil
}

func (h *ogenHandler) RevokeUserAPIToken(ctx context.Context, params gatewayapi.RevokeUserAPITokenParams) error {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}

	userID := params.UserId
	if userID == "self" {
		userID = actor
	}

	if err := h.metadataLifecycleClient.RevokeUserAPIToken(ctx, actor, userID); err != nil {
		return fmt.Errorf("revoke user API token: %w", err)
	}

	return nil
}

func (h *ogenHandler) RevokeSelfUserAPIToken(ctx context.Context) error {
	return h.RevokeUserAPIToken(ctx, gatewayapi.RevokeUserAPITokenParams{UserId: "self"})
}
