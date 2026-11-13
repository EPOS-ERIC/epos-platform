package clients

import (
	"context"

	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func (c *MetadataLifecycle) SyncOrganizations(ctx context.Context, values []domain.SyncedOrganization) error {
	request := &pb.SyncOrganizationsRequest{}
	for _, v := range values {
		organization := &pb.SyncOrganization{
			Nid:         v.NID,
			ParentNids:  v.ParentNIDs,
			Name:        v.Name,
			Street:      v.Street,
			Locality:    v.Locality,
			PostalCode:  v.PostalCode,
			Country:     v.Country,
			LogoUrl:     v.LogoURL,
			PageUrl:     v.PageURL,
			Email:       v.Email,
			Identifiers: organizationIdentifiers(v.Identifiers),
		}
		request.Organizations = append(request.Organizations, organization)
	}

	_, err := c.organizations.SyncOrganizations(ctx, request)
	if err != nil {
		return c.wrapError(ctx, "sync metadata-lifecycle organizations", err)
	}

	return nil
}
