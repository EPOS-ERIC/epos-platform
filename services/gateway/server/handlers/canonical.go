package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ListContactPoints(ctx context.Context) ([]gatewayapi.ContactPoint, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	v, err := h.metadataLifecycleClient.ListContactPoints(ctx, actor)
	if err != nil {
		return nil, fmt.Errorf("list contact points: %w", err)
	}
	out := []gatewayapi.ContactPoint{}
	for _, x := range v.ContactPoints {
		mapped, err := toAPIProtoContactPoint(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetContactPoint(ctx context.Context, q gatewayapi.GetContactPointParams) (*gatewayapi.ContactPoint, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	v, err := h.metadataLifecycleClient.GetContactPoint(ctx, actor, q.ContactPointId.String())
	if err != nil {
		return nil, fmt.Errorf("get contact point: %w", err)
	}
	r, err := toAPIProtoContactPoint(v.ContactPoint)
	if err != nil {
		return nil, fmt.Errorf("get contact point: %w", err)
	}
	return &r, nil
}

func (h *ogenHandler) ListOrganizations(ctx context.Context) ([]gatewayapi.CanonicalOrganization, error) {
	v, err := h.metadataLifecycleClient.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	out := []gatewayapi.CanonicalOrganization{}
	for _, x := range v.Organizations {
		mapped, err := toAPIProtoOrganization(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetOrganization(ctx context.Context, p gatewayapi.GetOrganizationParams) (*gatewayapi.CanonicalOrganization, error) {
	v, err := h.metadataLifecycleClient.GetOrganization(ctx, p.OrganizationId.String())
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	r, err := toAPIProtoOrganization(v.Organization)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return &r, nil
}

func (h *ogenHandler) CreateOrganization(ctx context.Context, q *gatewayapi.CreateOrganizationRequest) (*gatewayapi.CanonicalOrganization, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	v, err := h.metadataLifecycleClient.CreateOrganization(ctx, a, toProtoCreateOrganizationRequest(q))
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	r, err := toAPIProtoOrganization(v.Organization)
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	return &r, nil
}

func (h *ogenHandler) ReplaceOrganization(ctx context.Context, q *gatewayapi.ReplaceOrganizationRequest, p gatewayapi.ReplaceOrganizationParams) (*gatewayapi.CanonicalOrganization, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	in := toProtoReplaceOrganizationRequest(q, p.OrganizationId.String())
	v, err := h.metadataLifecycleClient.ReplaceOrganization(ctx, a, in)
	if err != nil {
		return nil, fmt.Errorf("replace organization: %w", err)
	}
	r, err := toAPIProtoOrganization(v.Organization)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) CreateContactPoint(ctx context.Context, q *gatewayapi.CreateContactPointRequest) (*gatewayapi.ContactPoint, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	v, err := h.metadataLifecycleClient.CreateContactPoint(ctx, a, &metadatalifecyclev1.CreateContactPointRequest{
		Email: string(q.Email),
		Role:  string(q.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("create contact point: %w", err)
	}
	r, err := toAPIProtoContactPoint(v.ContactPoint)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ReplaceContactPoint(ctx context.Context, q *gatewayapi.ReplaceContactPointRequest, p gatewayapi.ReplaceContactPointParams) (*gatewayapi.ContactPoint, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	v, err := h.metadataLifecycleClient.ReplaceContactPoint(ctx, a, &metadatalifecyclev1.ReplaceContactPointRequest{
		ContactPointId: p.ContactPointId.String(),
		Email:          string(q.Email),
		Role:           string(q.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("replace contact point: %w", err)
	}
	r, err := toAPIProtoContactPoint(v.ContactPoint)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) DeleteContactPoint(ctx context.Context, p gatewayapi.DeleteContactPointParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if _, err := h.metadataLifecycleClient.DeleteContactPoint(ctx, a, &metadatalifecyclev1.DeleteContactPointRequest{ContactPointId: p.ContactPointId.String()}); err != nil {
		return fmt.Errorf("delete contact point: %w", err)
	}
	return nil
}

func (h *ogenHandler) DeleteOrganization(ctx context.Context, p gatewayapi.DeleteOrganizationParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if _, err := h.metadataLifecycleClient.DeleteOrganization(ctx, a, &metadatalifecyclev1.DeleteOrganizationRequest{OrganizationId: p.OrganizationId.String()}); err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	return nil
}
