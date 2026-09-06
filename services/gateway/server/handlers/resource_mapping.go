package handlers

import (
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func toAPIResourceVersion(v *metadatalifecyclev1.ResourceVersion) (gatewayapi.ResourceVersion, error) {
	if v == nil {
		return gatewayapi.ResourceVersion{}, errors.New("decode metadata-lifecycle resource record: response is empty")
	}
	status, err := toAPIVersioningStatus(v.Status)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	resourceID, err := parseUUID("resource_version.resource_id", v.ResourceId)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	versionID, err := parseUUID("resource_version.version_id", v.VersionId)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	groupID, err := parseUUID("resource_version.group_id", v.GroupId)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	metadata, err := toAPIResourceMetadata(v)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	return gatewayapi.ResourceVersion{
		ResourceID: resourceID,
		VersionID:  versionID,
		Revision:   v.Revision,
		Status:     status,
		CreatedBy:  v.CreatedBy,
		GroupID:    groupID,
		GroupName:  v.GroupName,
		CreatedAt:  v.CreatedAt.AsTime(),
		UpdatedAt:  v.UpdatedAt.AsTime(),
		Metadata:   metadata,
	}, nil
}

func toAPIVersioningStatus(status metadatalifecyclev1.VersioningStatus) (gatewayapi.VersioningStatus, error) {
	switch status {
	case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_PUBLISHED:
		return gatewayapi.VersioningStatusPUBLISHED, nil
	case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_DRAFT:
		return gatewayapi.VersioningStatusDRAFT, nil
	case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_SUBMITTED:
		return gatewayapi.VersioningStatusSUBMITTED, nil
	case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_ARCHIVED:
		return gatewayapi.VersioningStatusARCHIVED, nil
	case metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_REJECTED:
		return gatewayapi.VersioningStatusREJECTED, nil
	default:
		return "", errors.New("unknown versioning status")
	}
}

func toAPIResourceMetadata(v *metadatalifecyclev1.ResourceVersion) (gatewayapi.ResourceVersionMetadata, error) {
	switch {
	case v.GetDataset() != nil:
		metadata, err := toAPIDatasetMetadata(v.GetDataset())
		if err != nil {
			return gatewayapi.ResourceVersionMetadata{}, err
		}
		return gatewayapi.NewDatasetResourceMetadataResourceVersionMetadata(metadata), nil
	case v.GetSoftwareApplication() != nil:
		metadata, mapErr := toAPISoftwareApplicationMetadata(v.GetSoftwareApplication())
		if mapErr != nil {
			return gatewayapi.ResourceVersionMetadata{}, mapErr
		}
		return gatewayapi.NewSoftwareApplicationResourceMetadataResourceVersionMetadata(metadata), nil
	case v.GetSoftwareSourceCode() != nil:
		metadata, mapErr := toAPISoftwareSourceCodeMetadata(v.GetSoftwareSourceCode())
		if mapErr != nil {
			return gatewayapi.ResourceVersionMetadata{}, mapErr
		}
		return gatewayapi.NewSoftwareSourceCodeResourceMetadataResourceVersionMetadata(metadata), nil
	default:
		return gatewayapi.ResourceVersionMetadata{}, errors.New("resource version metadata is nil or unsupported")
	}
}

func toAPIDatasetMetadata(v *metadatalifecyclev1.DatasetResourceMetadata) (gatewayapi.DatasetResourceMetadata, error) {
	id, err := parseUUID("dataset.id", v.GetId())
	if err != nil {
		return gatewayapi.DatasetResourceMetadata{}, err
	}
	result := gatewayapi.DatasetResourceMetadata{
		ResourceType:       gatewayapi.DatasetResourceMetadataResourceTypeDataset,
		ID:                 id,
		Title:              v.GetTitle(),
		Description:        v.GetDescription(),
		Keywords:           v.GetKeywords(),
		AccrualPeriodicity: toAPINilString(v.AccrualPeriodicity),
		QualityAnnotation:  toAPINilString(v.QualityAnnotation),
		AccessRight:        toAPINilString(v.AccessRight),
		SpatialCoverage:    toAPINilString(v.SpatialCoverage),
		Created:            toAPINilTime(v.Created),
		Updated:            toAPINilTime(v.Updated),
	}
	if v.TemporalCoverage != nil {
		result.TemporalCoverage = gatewayapi.NewOptTemporalCoverage(gatewayapi.TemporalCoverage{
			Start: toAPINilTime(v.TemporalCoverage.Start),
			End:   toAPINilTime(v.TemporalCoverage.End),
		})
	}
	for _, value := range v.PersistentIdentifiers {
		if value == nil {
			continue
		}
		mapped, err := toAPIIdentifier(value)
		if err != nil {
			return gatewayapi.DatasetResourceMetadata{}, err
		}
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, mapped)
	}
	for _, value := range v.ContactPoints {
		mapped, err := parseUUID("dataset.contact_point_id", value)
		if err != nil {
			return gatewayapi.DatasetResourceMetadata{}, err
		}
		result.ContactPoints = append(result.ContactPoints, mapped)
	}
	result.Providers, err = toAPIReferences(v.Providers, "dataset.provider_id")
	if err != nil {
		return gatewayapi.DatasetResourceMetadata{}, err
	}
	result.Categories, err = toAPIReferences(v.Categories, "dataset.category_id")
	if err != nil {
		return gatewayapi.DatasetResourceMetadata{}, err
	}
	for _, value := range v.Distributions {
		if value == nil {
			continue
		}
		mapped, err := toAPIDistribution(value)
		if err != nil {
			return gatewayapi.DatasetResourceMetadata{}, err
		}
		result.Distributions = append(result.Distributions, mapped)
	}
	return result, nil
}

func toAPISoftwareApplicationMetadata(v *metadatalifecyclev1.SoftwareApplicationResourceMetadata) (gatewayapi.SoftwareApplicationResourceMetadata, error) {
	id, err := parseUUID("software_application.id", v.GetId())
	if err != nil {
		return gatewayapi.SoftwareApplicationResourceMetadata{}, err
	}
	result := gatewayapi.SoftwareApplicationResourceMetadata{
		ResourceType:     gatewayapi.SoftwareApplicationResourceMetadataResourceTypeSoftwareApplication,
		ID:               id,
		Name:             v.GetName(),
		Description:      v.GetDescription(),
		MainEntityOfPage: toAPINilString(v.MainEntityOfPage),
		License:          toAPINilString(v.License),
		Version:          toAPINilString(v.Version),
		Keywords:         v.Keywords,
		DownloadURL:      toAPINilString(v.DownloadUrl),
		Requirements:     toAPINilString(v.Requirements),
		OperatingSystem:  toAPINilString(v.OperatingSystem),
	}
	result.Creators, err = toAPIReferences(v.Creators, "software_application.creator_id")
	if err != nil {
		return gatewayapi.SoftwareApplicationResourceMetadata{}, err
	}
	result.Categories, err = toAPIReferences(v.Categories, "software_application.category_id")
	if err != nil {
		return gatewayapi.SoftwareApplicationResourceMetadata{}, err
	}
	result.Identifiers, err = toAPIIdentifiers(v.Identifiers)
	if err != nil {
		return gatewayapi.SoftwareApplicationResourceMetadata{}, err
	}
	result.ContactPoints, err = toAPIUUIDs(v.ContactPoints, "software_application.contact_point_id")
	return result, err
}

func toAPISoftwareSourceCodeMetadata(v *metadatalifecyclev1.SoftwareSourceCodeResourceMetadata) (gatewayapi.SoftwareSourceCodeResourceMetadata, error) {
	id, err := parseUUID("software_source_code.id", v.GetId())
	if err != nil {
		return gatewayapi.SoftwareSourceCodeResourceMetadata{}, err
	}
	result := gatewayapi.SoftwareSourceCodeResourceMetadata{
		ResourceType:         gatewayapi.SoftwareSourceCodeResourceMetadataResourceTypeSoftwareSourceCode,
		ID:                   id,
		Name:                 v.GetName(),
		Description:          v.GetDescription(),
		ProgrammingLanguage:  toAPINilString(v.ProgrammingLanguage),
		Keywords:             v.Keywords,
		Repository:           toAPINilString(v.Repository),
		License:              toAPINilString(v.License),
		SoftwareRequirements: toAPINilString(v.SoftwareRequirements),
		RuntimePlatform:      toAPINilString(v.RuntimePlatform),
		MainEntityOfPage:     toAPINilString(v.MainEntityOfPage),
		Version:              toAPINilString(v.Version),
	}
	result.Creators, err = toAPIReferences(v.Creators, "software_source_code.creator_id")
	if err != nil {
		return gatewayapi.SoftwareSourceCodeResourceMetadata{}, err
	}
	result.Categories, err = toAPIReferences(v.Categories, "software_source_code.category_id")
	if err != nil {
		return gatewayapi.SoftwareSourceCodeResourceMetadata{}, err
	}
	result.Identifiers, err = toAPIIdentifiers(v.Identifiers)
	if err != nil {
		return gatewayapi.SoftwareSourceCodeResourceMetadata{}, err
	}
	result.ContactPoints, err = toAPIUUIDs(v.ContactPoints, "software_source_code.contact_point_id")
	return result, err
}

func toAPIReferences(values []*metadatalifecyclev1.ResourceReference, field string) ([]gatewayapi.ResourceVersionReference, error) {
	result := make([]gatewayapi.ResourceVersionReference, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		id, err := parseUUID(field, value.GetId())
		if err != nil {
			return nil, err
		}
		result = append(result, gatewayapi.ResourceVersionReference{
			ID:   id,
			Name: value.GetName(),
		})
	}
	return result, nil
}

func toAPIIdentifiers(values []*metadatalifecyclev1.ResourceIdentifier) ([]gatewayapi.ResourceIdentifier, error) {
	result := make([]gatewayapi.ResourceIdentifier, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		mapped, err := toAPIIdentifier(value)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func toAPIIdentifier(value *metadatalifecyclev1.ResourceIdentifier) (gatewayapi.ResourceIdentifier, error) {
	id, err := parseUUID("resource_identifier.id", value.GetId())
	if err != nil {
		return gatewayapi.ResourceIdentifier{}, err
	}
	return gatewayapi.ResourceIdentifier{
		ID:    id,
		Type:  value.GetType(),
		Value: value.GetValue(),
	}, nil
}

func toAPIUUIDs(values []string, field string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := parseUUID(field, value)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}
