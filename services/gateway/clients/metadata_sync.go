package clients

import (
	"context"

	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func (c *MetadataLifecycle) SyncMetadata(ctx context.Context, values []domain.SyncedOrganization, contacts []domain.SyncedContactPoint) error {
	request := &pb.SyncMetadataRequest{
		Organizations: &pb.OrganizationSnapshot{Items: make([]*pb.SyncOrganization, 0, len(values))},
		ContactPoints: &pb.ContactPointSnapshot{Items: make([]*pb.SyncContactPoint, 0, len(contacts))},
	}
	for _, v := range values {
		request.Organizations.Items = append(request.Organizations.Items, &pb.SyncOrganization{
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
		})
	}

	for _, contact := range contacts {
		request.ContactPoints.Items = append(request.ContactPoints.Items, &pb.SyncContactPoint{
			Nid:   contact.NID,
			Email: contact.Email,
		})
	}

	_, err := c.metadataSync.SyncMetadata(ctx, request)
	if err != nil {
		return c.wrapError(ctx, "apply metadata-lifecycle snapshots", err)
	}

	return nil
}
