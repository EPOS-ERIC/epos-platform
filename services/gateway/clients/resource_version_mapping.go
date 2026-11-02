package clients

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
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
	return domain.ResourceVersion{IsDataset: resource.GetDataset() != nil}, nil
}
