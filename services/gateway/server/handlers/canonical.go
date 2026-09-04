package handlers

import (
	"context"
	"fmt"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func (h *ogenHandler) ListContactPoints(ctx context.Context) ([]gatewayapi.ContactPoint, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.ListContactPoints(ctx, actor)
	if err != nil {
		return nil, fmt.Errorf("list contact points: %w", err)
	}
	out := []gatewayapi.ContactPoint{}
	for _, x := range v {
		mapped, err := toAPIContactPoint(x)
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
	v, err := h.metadataLifecycle.GetContactPoint(ctx, actor, q.ContactPointId.String())
	if err != nil {
		return nil, fmt.Errorf("get contact point: %w", err)
	}
	r, err := toAPIContactPoint(*v)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	return &r, nil
}

func (h *ogenHandler) ListOrganizations(ctx context.Context) ([]gatewayapi.CanonicalOrganization, error) {
	v, err := h.metadataLifecycle.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	out := []gatewayapi.CanonicalOrganization{}
	for _, x := range v {
		mapped, err := toAPIOrganization(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (h *ogenHandler) GetOrganization(ctx context.Context, p gatewayapi.GetOrganizationParams) (*gatewayapi.CanonicalOrganization, error) {
	v, err := h.metadataLifecycle.GetOrganization(ctx, p.OrganizationId.String())
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	r, err := toAPIOrganization(*v)
	if err != nil {
		return nil, fmt.Errorf("replace organization: %w", err)
	}
	return &r, nil
}

func (h *ogenHandler) CreateOrganization(ctx context.Context, q *gatewayapi.CreateOrganizationRequest) (*gatewayapi.CanonicalOrganization, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.CreateOrganization(ctx, a, toClientCreateOrganizationRequest(q))
	if err != nil {
		return nil, fmt.Errorf("create contact point: %w", err)
	}
	r, err := toAPIOrganization(*v)
	if err != nil {
		return nil, fmt.Errorf("replace contact point: %w", err)
	}
	return &r, nil
}

func (h *ogenHandler) ReplaceOrganization(ctx context.Context, q *gatewayapi.ReplaceOrganizationRequest, p gatewayapi.ReplaceOrganizationParams) (*gatewayapi.CanonicalOrganization, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	in := metadatalifecycleclient.ReplaceCanonicalOrganizationRequest{
		Name:            string(q.Name),
		Address:         toClientOrganizationAddress(q.Address),
		LogoUrl:         *metadatalifecycleclient.NewNullableString(toClientNilStringPtr(q.LogoURL)),
		PageUrl:         *metadatalifecycleclient.NewNullableString(toClientNilStringPtr(q.PageURL)),
		Email:           *metadatalifecycleclient.NewNullableString(toClientNilStringPtr(q.Email)),
		LeiCode:         *metadatalifecycleclient.NewNullableString(toClientNilStringPtr(q.LeiCode)),
		Parents:         make([]string, len(q.Parents)),
		ContactPointIds: make([]string, len(q.ContactPointIds)),
	}
	for i := range q.Parents {
		in.Parents[i] = q.Parents[i].String()
	}
	for i := range q.ContactPointIds {
		in.ContactPointIds[i] = q.ContactPointIds[i].String()
	}
	for _, x := range q.Identifiers {
		in.Identifiers = append(in.Identifiers, metadatalifecycleclient.CreateOrganizationIdentifierRequest{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	v, err := h.metadataLifecycle.ReplaceOrganization(ctx, a, p.OrganizationId.String(), in)
	if err != nil {
		return nil, fmt.Errorf("replace organization: %w", err)
	}
	r, err := toAPIOrganization(*v)
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
	v, err := h.metadataLifecycle.CreateContactPoint(ctx, a, metadatalifecycleclient.CreateContactPointRequest{
		Email: string(q.Email),
		Role:  string(q.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("create contact point: %w", err)
	}
	r, err := toAPIContactPoint(*v)
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
	v, err := h.metadataLifecycle.ReplaceContactPoint(ctx, a, p.ContactPointId.String(), metadatalifecycleclient.ReplaceContactPointRequest{
		Email: string(q.Email),
		Role:  string(q.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("replace contact point: %w", err)
	}
	r, err := toAPIContactPoint(*v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func toClientNullableString(v gatewayapi.OptNilString) metadatalifecycleclient.NullableString {
	return *metadatalifecycleclient.NewNullableString(toClientStringPtr(v))
}

func toClientNilString(v gatewayapi.NilString) metadatalifecycleclient.NullableString {
	if v.Null {
		return *metadatalifecycleclient.NewNullableString(nil)
	}
	return *metadatalifecycleclient.NewNullableString(&v.Value)
}

func toClientOrganizationAddress(v gatewayapi.OrganizationAddress) metadatalifecycleclient.CanonicalOrganizationAddress {
	return metadatalifecycleclient.CanonicalOrganizationAddress{
		Street:     toClientNilString(v.Street),
		Locality:   toClientNilString(v.Locality),
		PostalCode: toClientNilString(v.PostalCode),
		Country:    toClientNilString(v.Country),
	}
}

func toClientCreateOrganizationRequest(v *gatewayapi.CreateOrganizationRequest) metadatalifecycleclient.CreateCanonicalOrganizationRequest {
	r := metadatalifecycleclient.CreateCanonicalOrganizationRequest{
		Name:            string(v.Name),
		LogoUrl:         toClientNullableString(v.LogoURL),
		PageUrl:         toClientNullableString(v.PageURL),
		Email:           toClientNullableString(v.Email),
		LeiCode:         toClientNullableString(v.LeiCode),
		Parents:         make([]string, len(v.Parents)),
		ContactPointIds: make([]string, len(v.ContactPointIds)),
	}
	if v.Address.Set {
		a := v.Address.Value
		r.Address = &metadatalifecycleclient.CreateCanonicalOrganizationRequestAddress{
			Street:     toClientNilString(a.Street),
			Locality:   toClientNilString(a.Locality),
			PostalCode: toClientNilString(a.PostalCode),
			Country:    toClientNilString(a.Country),
		}
	}
	for i := range v.Parents {
		r.Parents[i] = v.Parents[i].String()
	}
	for i := range v.ContactPointIds {
		r.ContactPointIds[i] = v.ContactPointIds[i].String()
	}
	for _, x := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, metadatalifecycleclient.CreateOrganizationIdentifierRequest{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	return r
}

func (h *ogenHandler) DeleteContactPoint(ctx context.Context, p gatewayapi.DeleteContactPointParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.DeleteContactPoint(ctx, a, p.ContactPointId.String()); err != nil {
		return fmt.Errorf("delete contact point: %w", err)
	}
	return nil
}

func (h *ogenHandler) DeleteOrganization(ctx context.Context, p gatewayapi.DeleteOrganizationParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.DeleteOrganization(ctx, a, p.OrganizationId.String()); err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	return nil
}
