package clients

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

func toProtoResourceType(resourceType domain.ResourceType) (metadatalifecyclev1.ResourceType, error) {
	switch resourceType {
	case domain.ResourceTypeDataset:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET, nil
	case domain.ResourceTypeSoftwareApplication:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION, nil
	case domain.ResourceTypeSoftwareSourceCode:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE, nil
	default:
		return 0, fmt.Errorf("unknown resource type %q", resourceType)
	}
}

func fromProtoResourceVersion(resource *metadatalifecyclev1.ResourceVersion) (domain.ResourceVersion, error) {
	if resource == nil {
		return domain.ResourceVersion{}, fmt.Errorf("metadata-lifecycle returned an empty resource version")
	}
	result := domain.ResourceVersion{}
	if dataset := resource.GetDataset(); dataset != nil {
		result.IsDataset = true
		for _, distribution := range dataset.GetDistributions() {
			if distribution == nil || distribution.GetWebService() == nil {
				continue
			}
			id, err := uuid.Parse(distribution.GetId())
			if err != nil {
				return domain.ResourceVersion{}, fmt.Errorf("invalid distribution ID: %w", err)
			}
			result.Distributions = append(result.Distributions, domain.ResourceDistribution{ID: id})
		}
	}
	return result, nil
}
