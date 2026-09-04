package clients

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func (c *MetadataLifecycle) ListContactPoints(ctx context.Context, actorID string) ([]client.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return nil, fmt.Errorf("metadata-lifecycle contact-point client is not configured")
	}

	points, response, err := c.contactPoints.ListContactPoints(ctx).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapCanonicalError("list metadata-lifecycle contact points", response, err)
	}

	return points, nil
}

func (c *MetadataLifecycle) GetContactPoint(ctx context.Context, actorID, id string) (*client.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return nil, fmt.Errorf("metadata-lifecycle contact-point client is not configured")
	}

	point, response, err := c.contactPoints.GetContactPoint(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapCanonicalError("get metadata-lifecycle contact point", response, err)
	}
	if point == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle contact point: response is empty")
	}

	return point, nil
}

func (c *MetadataLifecycle) CreateContactPoint(ctx context.Context, actorID string, request client.CreateContactPointRequest) (*client.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return nil, fmt.Errorf("metadata-lifecycle contact-point client is not configured")
	}

	point, response, err := c.contactPoints.CreateContactPoint(ctx).
		XEPOSUserID(actorID).
		CreateContactPointRequest(request).Execute()
	if err != nil {
		return nil, wrapCanonicalError("create metadata-lifecycle contact point", response, err)
	}
	if point == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle contact point: response is empty")
	}

	return point, nil
}

func (c *MetadataLifecycle) ReplaceContactPoint(ctx context.Context, actorID, id string, request client.ReplaceContactPointRequest) (*client.ContactPoint, error) {
	if c == nil || c.contactPoints == nil {
		return nil, fmt.Errorf("metadata-lifecycle contact-point client is not configured")
	}

	point, response, err := c.contactPoints.ReplaceContactPoint(ctx, id).
		XEPOSUserID(actorID).
		ReplaceContactPointRequest(request).Execute()
	if err != nil {
		return nil, wrapCanonicalError("replace metadata-lifecycle contact point", response, err)
	}
	if point == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle contact-point: response is empty")
	}

	return point, nil
}

func (c *MetadataLifecycle) DeleteContactPoint(ctx context.Context, actorID, id string) error {
	if c == nil || c.contactPoints == nil {
		return fmt.Errorf("metadata-lifecycle contact-point client is not configured")
	}

	response, err := c.contactPoints.DeleteContactPoint(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapCanonicalError("delete metadata-lifecycle contact point", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) ListOrganizations(ctx context.Context) ([]client.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return nil, fmt.Errorf("metadata-lifecycle organization client is not configured")
	}

	organizations, response, err := c.organizations.ListOrganizations(ctx).Execute()
	if err != nil {
		return nil, wrapCanonicalError("list metadata-lifecycle organizations", response, err)
	}

	return organizations, nil
}

func (c *MetadataLifecycle) GetOrganization(ctx context.Context, id string) (*client.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return nil, fmt.Errorf("metadata-lifecycle organization client is not configured")
	}

	organization, response, err := c.organizations.GetOrganization(ctx, id).Execute()
	if err != nil {
		return nil, wrapCanonicalError("get metadata-lifecycle organization", response, err)
	}
	if organization == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle organization: response is empty")
	}

	return organization, nil
}

func (c *MetadataLifecycle) CreateOrganization(ctx context.Context, actorID string, request client.CreateCanonicalOrganizationRequest) (*client.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return nil, fmt.Errorf("metadata-lifecycle organization client is not configured")
	}

	organization, response, err := c.organizations.CreateOrganization(ctx).
		XEPOSUserID(actorID).
		CreateCanonicalOrganizationRequest(request).
		Execute()
	if err != nil {
		return nil, wrapCanonicalError("create metadata-lifecycle organization", response, err)
	}
	if organization == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle organization: response is empty")
	}
	return organization, nil
}

func (c *MetadataLifecycle) ReplaceOrganization(ctx context.Context, actorID, id string, request client.ReplaceCanonicalOrganizationRequest) (*client.CanonicalOrganization, error) {
	if c == nil || c.organizations == nil {
		return nil, fmt.Errorf("metadata-lifecycle organization client is not configured")
	}

	organization, response, err := c.organizations.ReplaceOrganization(ctx, id).
		XEPOSUserID(actorID).
		ReplaceCanonicalOrganizationRequest(request).
		Execute()
	if err != nil {
		return nil, wrapCanonicalError("replace metadata-lifecycle organization", response, err)
	}
	if organization == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle organization: response is empty")
	}
	return organization, nil
}

func (c *MetadataLifecycle) DeleteOrganization(ctx context.Context, actorID, id string) error {
	if c == nil || c.organizations == nil {
		return fmt.Errorf("metadata-lifecycle organization client is not configured")
	}

	response, err := c.organizations.DeleteOrganization(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapCanonicalError("delete metadata-lifecycle organization", response, err)
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
