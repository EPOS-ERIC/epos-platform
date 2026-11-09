package grpcapi

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/contactrequests"
)

func toDomainContactRequestTarget(value *metadatalifecyclev1.PrepareEmailsRequest) (contactrequests.Target, error) {
	switch target := value.GetTarget().(type) {
	case *metadatalifecyclev1.PrepareEmailsRequest_Resource:
		if target.Resource == nil {
			return contactrequests.Target{}, apperror.Field("resource", "Resource target is required.", nil)
		}

		resourceType, ok := toDomainResourceType(target.Resource.GetResourceType())
		if !ok {
			return contactrequests.Target{}, apperror.Field("resource.resource_type", "Resource type is not supported.", nil)
		}

		resourceID, err := parseUUIDAt(target.Resource.GetResourceId(), "resource id", "resource.resource_id")
		if err != nil {
			return contactrequests.Target{}, err
		}

		return contactrequests.Target{ResourceType: resourceType, ResourceID: resourceID}, nil
	case *metadatalifecyclev1.PrepareEmailsRequest_DistributionId:
		distributionID, err := parseUUIDAt(target.DistributionId, "distribution id", "distribution_id")
		if err != nil {
			return contactrequests.Target{}, err
		}

		return contactrequests.Target{DistributionID: distributionID}, nil
	default:
		return contactrequests.Target{}, apperror.Field("target", "A resource or distribution target is required.", nil)
	}
}

func toProtoPreparedContactEmails(values []contactrequests.PreparedEmail) []*metadatalifecyclev1.PreparedContactEmail {
	result := make([]*metadatalifecyclev1.PreparedContactEmail, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.PreparedContactEmail{
			TargetEmail: value.TargetEmail,
			Subject:     value.Subject,
			Body:        value.Body,
		}
	}

	return result
}
