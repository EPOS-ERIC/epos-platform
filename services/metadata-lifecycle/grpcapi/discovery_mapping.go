package grpcapi

import (
	"errors"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrInvalidDiscoveryFilter = errors.New("invalid discovery filter")

func toDomainDiscoveryFilter(value *metadatalifecyclev1.SearchResourcesRequest) (domain.DiscoveryFilter, error) {
	query, err := optionalDiscoveryString(value.Query, "query")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	groupID, err := optionalDiscoveryUUID(value.GroupId, "group id")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	spatialFilter, err := optionalDiscoverySpatialFilter(value.SpatialFilter)
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	dateFrom, err := timestampInput(value.DateFrom)
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	dateTo, err := timestampInput(value.DateTo)
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	providers, err := resourceParseUUIDs(value.ProviderIds, "provider id")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}

	resourceTypes, err := toDomainDiscoveryResourceTypes(value.ResourceTypes)
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	concepts, err := toDomainDiscoveryConcepts(value.Concepts)
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	statuses, err := toDomainDiscoveryStatuses(value.VersioningStatuses)
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	editorID, err := optionalDiscoveryString(value.EditorId, "editor id")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}

	filter, err := domain.NewDiscoveryFilter(domain.DiscoveryFilter{
		Query:              query,
		Types:              resourceTypes,
		Concepts:           concepts,
		GroupID:            groupID,
		SpatialFilter:      spatialFilter,
		DateFrom:           dateFrom,
		DateTo:             dateTo,
		Providers:          providers,
		Keywords:           value.Keywords,
		EditorID:           editorID,
		VersioningStatuses: statuses,
	})
	if err != nil {
		return domain.DiscoveryFilter{}, discoveryInvalidError(err)
	}
	return filter, nil
}

func optionalDiscoveryString(value *string, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	if *value == "" {
		return nil, status.Errorf(codes.InvalidArgument, "%s must not be empty", field)
	}
	return value, nil
}

func optionalDiscoveryUUID(value *string, field string) (*uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	id, err := uuid.Parse(*value)
	if err != nil || id == uuid.Nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid %s", field)
	}
	return &id, nil
}

func optionalDiscoverySpatialFilter(value *string) (geom.T, error) {
	if value == nil {
		return nil, nil
	}
	filter, err := domain.NewSpatialFilter(*value)
	if err != nil {
		return nil, discoveryInvalidError(err)
	}
	return filter, nil
}

func toDomainDiscoveryResourceTypes(values []metadatalifecyclev1.ResourceType) ([]domain.ResourceType, error) {
	result := make([]domain.ResourceType, 0, len(values))
	for _, value := range values {
		switch value {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			result = append(result, domain.Dataset)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			result = append(result, domain.SoftwareApplication)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			result = append(result, domain.SoftwareSourceCode)
		default:
			return nil, discoveryInvalidError(fmt.Errorf("unsupported resource type %q", value))
		}
	}
	return result, nil
}

func toDomainDiscoveryConcepts(values []metadatalifecyclev1.TopLevelConcept) ([]domain.TopLevelConcept, error) {
	result := make([]domain.TopLevelConcept, 0, len(values))
	for _, value := range values {
		switch value {
		case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA:
			result = append(result, domain.TopLevelConceptData)
		case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES:
			result = append(result, domain.TopLevelConceptFacilities)
		case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE:
			result = append(result, domain.TopLevelConceptSoftware)
		default:
			return nil, discoveryInvalidError(fmt.Errorf("unsupported discovery concept %q", value))
		}
	}
	return result, nil
}

func toDomainDiscoveryStatuses(values []metadatalifecyclev1.VersioningStatus) ([]domain.Status, error) {
	result := make([]domain.Status, 0, len(values))
	for _, value := range values {
		switch value {
		case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_PUBLISHED:
			result = append(result, domain.Published)
		case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_DRAFT:
			result = append(result, domain.Draft)
		case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_SUBMITTED:
			result = append(result, domain.Submitted)
		case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_ARCHIVED:
			result = append(result, domain.Archived)
		case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_REJECTED:
			result = append(result, domain.Rejected)
		default:
			return nil, discoveryInvalidError(fmt.Errorf("unsupported discovery status %q", value))
		}
	}
	return result, nil
}

func discoveryInvalidError(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidDiscoveryFilter, err)
}

func toProtoDiscoveryResponse(values []domain.DiscoveryItem) (*metadatalifecyclev1.SearchResourcesResponse, error) {
	items := make([]*metadatalifecyclev1.DiscoveryItem, 0, len(values))
	for _, value := range values {
		item, err := toProtoDiscoveryItem(value)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &metadatalifecyclev1.SearchResourcesResponse{Items: items}, nil
}

func toProtoDiscoveryItem(value domain.DiscoveryItem) (*metadatalifecyclev1.DiscoveryItem, error) {
	resourceID := value.ResourceID.String()
	versionID := value.VersionID.String()
	name := value.Name
	resourceStatus := toGRPCVersioningStatus(value.Status)
	updatedAt := timestamppb.New(value.UpdatedAt)
	concept := toProtoDiscoveryConcept(value.Concept)
	createdBy := toProtoDiscoveryCreatedBy(value.CreatedBy)
	group := toProtoDiscoveryReference(value.Group)
	switch value.Type {
	case domain.Dataset:
		return &metadatalifecyclev1.DiscoveryItem{
			Resource: &metadatalifecyclev1.DiscoveryItem_Dataset{
				Dataset: &metadatalifecyclev1.DatasetDiscoveryItem{
					ResourceId:    resourceID,
					VersionId:     versionID,
					Name:          name,
					Status:        resourceStatus,
					UpdatedAt:     updatedAt,
					Concept:       concept,
					CreatedBy:     createdBy,
					Group:         group,
					Providers:     toProtoDiscoveryReferences(value.Providers),
					Categories:    toProtoDiscoveryReferences(value.Categories),
					Distributions: toProtoDiscoveryReferences(value.Distributions),
				},
			},
		}, nil
	case domain.SoftwareApplication:
		return &metadatalifecyclev1.DiscoveryItem{
			Resource: &metadatalifecyclev1.DiscoveryItem_SoftwareApplication{
				SoftwareApplication: &metadatalifecyclev1.SoftwareApplicationDiscoveryItem{
					ResourceId: resourceID,
					VersionId:  versionID,
					Name:       name,
					Status:     resourceStatus,
					UpdatedAt:  updatedAt,
					Concept:    concept,
					CreatedBy:  createdBy,
					Group:      group,
					Creators:   toProtoDiscoveryReferences(value.Creators),
					Categories: toProtoDiscoveryReferences(value.Categories),
				},
			},
		}, nil
	case domain.SoftwareSourceCode:
		return &metadatalifecyclev1.DiscoveryItem{
			Resource: &metadatalifecyclev1.DiscoveryItem_SoftwareSourceCode{
				SoftwareSourceCode: &metadatalifecyclev1.SoftwareSourceCodeDiscoveryItem{
					ResourceId: resourceID,
					VersionId:  versionID,
					Name:       name,
					Status:     resourceStatus,
					UpdatedAt:  updatedAt,
					Concept:    concept,
					CreatedBy:  createdBy,
					Group:      group,
					Creators:   toProtoDiscoveryReferences(value.Creators),
					Categories: toProtoDiscoveryReferences(value.Categories),
				},
			},
		}, nil
	default:
		return nil, discoveryInvalidError(fmt.Errorf("unsupported resource type %q", value.Type))
	}
}

func toProtoDiscoveryConcept(value domain.TopLevelConcept) metadatalifecyclev1.TopLevelConcept {
	switch value {
	case domain.TopLevelConceptData:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA
	case domain.TopLevelConceptFacilities:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES
	case domain.TopLevelConceptSoftware:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE
	default:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_UNSPECIFIED
	}
}

func toProtoDiscoveryCreatedBy(value *domain.DiscoveryCreatedBy) *metadatalifecyclev1.UserReference {
	if value == nil {
		return nil
	}
	return &metadatalifecyclev1.UserReference{Id: value.ID, Name: value.Name}
}

func toProtoDiscoveryReference(value domain.DiscoveryReference) *metadatalifecyclev1.EntityReference {
	return &metadatalifecyclev1.EntityReference{Id: value.ID.String(), Name: value.Name}
}

func toProtoDiscoveryReferences(values []domain.DiscoveryReference) []*metadatalifecyclev1.EntityReference {
	result := make([]*metadatalifecyclev1.EntityReference, 0, len(values))
	for _, value := range values {
		result = append(result, toProtoDiscoveryReference(value))
	}
	return result
}
