//nolint:dupl,exhaustruct_v5,wrapcheck
package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ListContactPoints(ctx context.Context) ([]gatewayapi.ContactPoint, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.canonical.ListContactPoints(ctx, actor)
	if err != nil {
		return nil, err
	}
	out := []gatewayapi.ContactPoint{}
	for _, x := range v {
		mapped, err := mapContact(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetContactPoint(ctx context.Context, q gatewayapi.GetContactPointParams) (*gatewayapi.ContactPoint, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.canonical.GetContactPoint(ctx, actor, q.ContactPointId.String())
	if err != nil {
		return nil, err
	}
	r, err := mapContact(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ListOrganizations(ctx context.Context) ([]gatewayapi.CanonicalOrganization, error) {
	v, err := h.canonical.ListOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	out := []gatewayapi.CanonicalOrganization{}
	for _, x := range v {
		mapped, err := mapOrganization(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetOrganization(ctx context.Context, p gatewayapi.GetOrganizationParams) (*gatewayapi.CanonicalOrganization, error) {
	v, err := h.canonical.GetOrganization(ctx, p.OrganizationId.String())
	if err != nil {
		return nil, err
	}
	r, err := mapOrganization(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) CreateOrganization(ctx context.Context, q *gatewayapi.CreateOrganizationRequest) (*gatewayapi.CanonicalOrganization, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.canonical.CreateOrganization(ctx, a, mapCreateOrganization(q))
	if err != nil {
		return nil, err
	}
	r, err := mapOrganization(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ReplaceOrganization(ctx context.Context, q *gatewayapi.ReplaceOrganizationRequest, p gatewayapi.ReplaceOrganizationParams) (*gatewayapi.CanonicalOrganization, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	in := domain.ReplaceOrganizationInput{Name: string(q.Name), Address: mapAddress(q.Address), LogoURL: fromNilString(q.LogoURL), PageURL: fromNilString(q.PageURL), Email: fromNilString(q.Email), LEICode: fromNilString(q.LeiCode), Parents: make([]string, len(q.Parents)), ContactPointIDs: make([]string, len(q.ContactPointIds))}
	for i := range q.Parents {
		in.Parents[i] = q.Parents[i].String()
	}
	for i := range q.ContactPointIds {
		in.ContactPointIDs[i] = q.ContactPointIds[i].String()
	}
	for _, x := range q.Identifiers {
		in.Identifiers = append(in.Identifiers, domain.OrganizationIdentifierInput{Type: string(x.Type), Value: string(x.Value)})
	}
	v, err := h.canonical.ReplaceOrganization(ctx, a, p.OrganizationId.String(), in)
	if err != nil {
		return nil, err
	}
	r, err := mapOrganization(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) CreateContactPoint(ctx context.Context, q *gatewayapi.CreateContactPointRequest) (*gatewayapi.ContactPoint, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.canonical.CreateContactPoint(ctx, a, domain.CreateContactPointInput{Email: string(q.Email), Role: string(q.Role)})
	if err != nil {
		return nil, err
	}
	r, err := mapContact(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) ReplaceContactPoint(ctx context.Context, q *gatewayapi.ReplaceContactPointRequest, p gatewayapi.ReplaceContactPointParams) (*gatewayapi.ContactPoint, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.canonical.ReplaceContactPoint(ctx, a, p.ContactPointId.String(), domain.ReplaceContactPointInput{Email: string(q.Email), Role: string(q.Role)})
	if err != nil {
		return nil, err
	}
	r, err := mapContact(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) DeleteContactPoint(ctx context.Context, p gatewayapi.DeleteContactPointParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return h.canonical.DeleteContactPoint(ctx, a, p.ContactPointId.String())
}

func (h *ogenHandler) DeleteOrganization(ctx context.Context, p gatewayapi.DeleteOrganizationParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return h.canonical.DeleteOrganization(ctx, a, p.OrganizationId.String())
}
