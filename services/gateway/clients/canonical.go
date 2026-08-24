package clients

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
)

func (c *Backoffice) ListContactPoints(ctx context.Context, actorID string) ([]domain.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return nil, fmt.Errorf("backoffice contact-point client is not configured")
	}

	points, response, err := c.contactPoints.ListContactPoints(ctx).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapCanonicalError("list backoffice contact points", response, err)
	}

	result := make([]domain.ContactPoint, 0, len(points))
	for _, point := range points {
		result = append(result, toDomainContactPoint(point))
	}

	return result, nil
}

func (c *Backoffice) GetContactPoint(ctx context.Context, actorID, id string) (domain.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return domain.ContactPoint{}, fmt.Errorf("backoffice contact-point client is not configured")
	}

	point, response, err := c.contactPoints.GetContactPoint(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return domain.ContactPoint{}, wrapCanonicalError("get backoffice contact point", response, err)
	}
	if point == nil {
		return domain.ContactPoint{}, fmt.Errorf("decode backoffice contact point: response is empty")
	}

	return toDomainContactPoint(*point), nil
}

func (c *Backoffice) CreateContactPoint(ctx context.Context, actorID string, input domain.CreateContactPointInput) (domain.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return domain.ContactPoint{}, fmt.Errorf("backoffice contact-point client is not configured")
	}

	point, response, err := c.contactPoints.CreateContactPoint(ctx).
		XEPOSUserID(actorID).
		CreateContactPointRequest(client.CreateContactPointRequest{
			SourceIri: input.SourceIRI,
			Email:     input.Email,
			Role:      input.Role,
		}).Execute()
	if err != nil {
		return domain.ContactPoint{}, wrapCanonicalError("create backoffice contact point", response, err)
	}
	if point == nil {
		return domain.ContactPoint{}, fmt.Errorf("decode backoffice contact point: response is empty")
	}

	return toDomainContactPoint(*point), nil
}

func (c *Backoffice) ReplaceContactPoint(ctx context.Context, actorID, id string, input domain.ReplaceContactPointInput) (domain.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return domain.ContactPoint{}, fmt.Errorf("backoffice contact-point client is not configured")
	}

	point, response, err := c.contactPoints.ReplaceContactPoint(ctx, id).
		XEPOSUserID(actorID).
		ReplaceContactPointRequest(client.ReplaceContactPointRequest{
			Email: input.Email,
			Role:  input.Role,
		}).Execute()
	if err != nil {
		return domain.ContactPoint{}, wrapCanonicalError("replace backoffice contact point", response, err)
	}
	if point == nil {
		return domain.ContactPoint{}, fmt.Errorf("decode backoffice contact point: response is empty")
	}

	return toDomainContactPoint(*point), nil
}

func (c *Backoffice) DeleteContactPoint(ctx context.Context, actorID, id string) error {
	if c == nil || c.contactPoints == nil {
		return fmt.Errorf("backoffice contact-point client is not configured")
	}

	response, err := c.contactPoints.DeleteContactPoint(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapCanonicalError("delete backoffice contact point", response, err)
	}

	return nil
}

func (c *Backoffice) ListOrganizations(ctx context.Context) ([]domain.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return nil, fmt.Errorf("backoffice organization client is not configured")
	}

	organizations, response, err := c.organizations.ListOrganizations(ctx).Execute()
	if err != nil {
		return nil, wrapCanonicalError("list backoffice organizations", response, err)
	}

	result := make([]domain.CanonicalOrganization, 0, len(organizations))
	for _, organization := range organizations {
		result = append(result, toDomainCanonicalOrganization(organization))
	}

	return result, nil
}

func (c *Backoffice) GetOrganization(ctx context.Context, id string) (domain.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("backoffice organization client is not configured")
	}

	organization, response, err := c.organizations.GetOrganization(ctx, id).Execute()
	if err != nil {
		return domain.CanonicalOrganization{}, wrapCanonicalError("get backoffice organization", response, err)
	}
	if organization == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("decode backoffice organization: response is empty")
	}

	return toDomainCanonicalOrganization(*organization), nil
}

func (c *Backoffice) CreateOrganization(ctx context.Context, actorID string, input domain.CreateOrganizationInput) (domain.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("backoffice organization client is not configured")
	}

	request := client.CreateCanonicalOrganizationRequest{
		SourceIri:       input.SourceIRI,
		Name:            input.Name,
		Address:         nil,
		LogoUrl:         client.NullableString{},
		PageUrl:         client.NullableString{},
		Email:           client.NullableString{},
		LeiCode:         client.NullableString{},
		Identifiers:     toClientIdentifierInputs(input.Identifiers),
		Parents:         input.Parents,
		ContactPointIds: input.ContactPointIDs,
	}
	if input.Address != nil {
		request.Address = newClientCreateAddress(*input.Address)
	}
	setNullable(&request.LogoUrl, input.LogoURL)
	setNullable(&request.PageUrl, input.PageURL)
	setNullable(&request.Email, input.Email)
	setNullable(&request.LeiCode, input.LEICode)

	organization, response, err := c.organizations.CreateOrganization(ctx).
		XEPOSUserID(actorID).
		CreateCanonicalOrganizationRequest(request).
		Execute()
	if err != nil {
		return domain.CanonicalOrganization{}, wrapCanonicalError("create backoffice organization", response, err)
	}
	if organization == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("decode backoffice organization: response is empty")
	}
	return toDomainCanonicalOrganization(*organization), nil
}

func (c *Backoffice) ReplaceOrganization(ctx context.Context, actorID, id string, input domain.ReplaceOrganizationInput) (domain.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("backoffice organization client is not configured")
	}

	request := client.ReplaceCanonicalOrganizationRequest{
		Name:            input.Name,
		Address:         newClientAddress(input.Address),
		LogoUrl:         client.NullableString{},
		PageUrl:         client.NullableString{},
		Email:           client.NullableString{},
		LeiCode:         client.NullableString{},
		Identifiers:     toClientIdentifierInputs(input.Identifiers),
		Parents:         input.Parents,
		ContactPointIds: input.ContactPointIDs,
	}
	setNullable(&request.LogoUrl, input.LogoURL)
	setNullable(&request.PageUrl, input.PageURL)
	setNullable(&request.Email, input.Email)
	setNullable(&request.LeiCode, input.LEICode)

	organization, response, err := c.organizations.ReplaceOrganization(ctx, id).
		XEPOSUserID(actorID).
		ReplaceCanonicalOrganizationRequest(request).
		Execute()
	if err != nil {
		return domain.CanonicalOrganization{}, wrapCanonicalError("replace backoffice organization", response, err)
	}
	if organization == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("decode backoffice organization: response is empty")
	}
	return toDomainCanonicalOrganization(*organization), nil
}

func (c *Backoffice) DeleteOrganization(ctx context.Context, actorID, id string) error {
	if c == nil || c.organizations == nil {
		return fmt.Errorf("backoffice organization client is not configured")
	}

	response, err := c.organizations.DeleteOrganization(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapCanonicalError("delete backoffice organization", response, err)
	}
	return nil
}

func toDomainContactPoint(point client.ContactPoint) domain.ContactPoint {
	return domain.ContactPoint{ID: point.Id, SourceIRI: point.SourceIri, Email: point.Email, Role: point.Role}
}

func toDomainCanonicalOrganization(organization client.CanonicalOrganization) domain.CanonicalOrganization {
	result := domain.CanonicalOrganization{
		ID:              organization.Id,
		SourceIRI:       organization.SourceIri,
		Name:            organization.Name,
		Address:         domain.OrganizationAddress{Street: organization.Address.Street.Get(), Locality: organization.Address.Locality.Get(), PostalCode: organization.Address.PostalCode.Get(), Country: organization.Address.Country.Get()},
		LogoURL:         organization.LogoUrl.Get(),
		PageURL:         organization.PageUrl.Get(),
		Email:           organization.Email.Get(),
		LEICode:         organization.LeiCode.Get(),
		Parents:         append([]string{}, organization.Parents...),
		ContactPointIDs: append([]string{}, organization.ContactPointIds...),
		Identifiers:     make([]domain.OrganizationIdentifier, 0, len(organization.Identifiers)),
	}
	for _, identifier := range organization.Identifiers {
		result.Identifiers = append(result.Identifiers, domain.OrganizationIdentifier{ID: identifier.Id, SourceIRI: identifier.SourceIri, Type: identifier.Type, Value: identifier.Value})
	}
	return result
}

func wrapCanonicalError(operation string, response *http.Response, err error) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("%s: %w", operation, domain.ErrBadRequest)
		case http.StatusForbidden:
			return fmt.Errorf("%s: %w", operation, domain.ErrForbidden)
		case http.StatusNotFound:
			return fmt.Errorf("%s: %w", operation, domain.ErrNotFound)
		case http.StatusConflict:
			return fmt.Errorf("%s: %w", operation, domain.ErrConflict)
		}
	}
	return wrapConnectionError(operation, err)
}

func toClientIdentifierInputs(inputs []domain.OrganizationIdentifierInput) []client.CreateOrganizationIdentifierRequest {
	result := make([]client.CreateOrganizationIdentifierRequest, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, client.CreateOrganizationIdentifierRequest{SourceIri: input.SourceIRI, Type: input.Type, Value: input.Value})
	}
	return result
}

func newClientCreateAddress(address domain.OrganizationAddress) *client.CreateCanonicalOrganizationRequestAddress {
	result := &client.CreateCanonicalOrganizationRequestAddress{
		Street:     client.NullableString{},
		Locality:   client.NullableString{},
		PostalCode: client.NullableString{},
		Country:    client.NullableString{},
	}
	setNullable(&result.Street, address.Street)
	setNullable(&result.Locality, address.Locality)
	setNullable(&result.PostalCode, address.PostalCode)
	setNullable(&result.Country, address.Country)
	return result
}

func newClientAddress(address domain.OrganizationAddress) client.CanonicalOrganizationAddress {
	result := client.CanonicalOrganizationAddress{
		Street:     client.NullableString{},
		Locality:   client.NullableString{},
		PostalCode: client.NullableString{},
		Country:    client.NullableString{},
	}
	setNullable(&result.Street, address.Street)
	setNullable(&result.Locality, address.Locality)
	setNullable(&result.PostalCode, address.PostalCode)
	setNullable(&result.Country, address.Country)
	return result
}

func setNullable(value *client.NullableString, input *string) {
	if input == nil {
		value.Set(nil)
		return
	}
	value.Set(input)
}
