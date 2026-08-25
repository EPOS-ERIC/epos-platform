package clients

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func (c *Backoffice) ListContactPoints(ctx context.Context, actorID string) ([]domain.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return nil, fmt.Errorf("backoffice contact-point client is not configured")
	}

	points, response, err := c.contactPoints.ListContactPoints(ctx).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapCanonicalError("list backoffice contact points", response, err)
	}

	return mapSlice(points, toDomainContactPoint), nil
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
		CreateContactPointRequest(toClientCreateContactPointRequest(input)).Execute()
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
		ReplaceContactPointRequest(toClientReplaceContactPointRequest(input)).Execute()
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

	return mapSlice(organizations, toDomainCanonicalOrganization), nil
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

	request := toClientCreateCanonicalOrganizationRequest(input)

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

	request := toClientReplaceCanonicalOrganizationRequest(input)

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
