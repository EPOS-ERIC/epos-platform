package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) PrepareResourceContactRequestEmails(ctx context.Context, actorID string, resourceType domain.ResourceType, resourceID uuid.UUID, message string) ([]domain.ContactRequestEmail, error) {
	protoResourceType, err := toProtoResourceType(resourceType)
	if err != nil {
		return nil, err
	}

	return c.prepareContactRequestEmails(ctx, actorID, &metadatalifecyclev1.PrepareEmailsRequest{
		Target: &metadatalifecyclev1.PrepareEmailsRequest_Resource{
			Resource: &metadatalifecyclev1.ContactRequestResource{
				ResourceType: protoResourceType,
				ResourceId:   resourceID.String(),
			},
		},
		Message: message,
	})
}

func (c *MetadataLifecycle) PrepareDistributionContactRequestEmails(ctx context.Context, actorID string, distributionID uuid.UUID, message string) ([]domain.ContactRequestEmail, error) {
	return c.prepareContactRequestEmails(ctx, actorID, &metadatalifecyclev1.PrepareEmailsRequest{
		Target:  &metadatalifecyclev1.PrepareEmailsRequest_DistributionId{DistributionId: distributionID.String()},
		Message: message,
	})
}

func (c *MetadataLifecycle) prepareContactRequestEmails(ctx context.Context, actorID string, request *metadatalifecyclev1.PrepareEmailsRequest) ([]domain.ContactRequestEmail, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactRequests.PrepareEmails(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "prepare contact request emails", err)
	}

	if response == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle contact request emails: response is empty")
	}

	emails, err := mapSliceErr(response.GetEmails(), fromProtoPreparedContactEmail)
	if err != nil {
		return nil, fmt.Errorf("decode metadata-lifecycle contact request emails: %w", err)
	}

	return emails, nil
}
