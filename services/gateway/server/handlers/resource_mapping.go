package handlers

import (
	"errors"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
	"github.com/google/uuid"
)

func toAPIResourceVersion(v *metadatalifecycleclient.ResourceRecord) (gatewayapi.ResourceVersion, error) {
	if v == nil {
		return gatewayapi.ResourceVersion{}, errors.New("decode metadata-lifecycle resource record: response is empty")
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
	metadata, err := toAPIResourceMetadata(v.Metadata)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	return gatewayapi.ResourceVersion{
		ResourceID: resourceID,
		VersionID:  versionID,
		Revision:   v.Revision,
		Status:     string(v.Status),
		CreatedBy:  v.CreatedBy,
		GroupID:    groupID,
		GroupName:  v.GroupName,
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
		Metadata:   metadata,
	}, nil
}

func toAPIResourceMetadata(v metadatalifecycleclient.ResourceRecordMetadata) (gatewayapi.ResourceVersionMetadata, error) {
	switch {
	case v.DatasetMetadata != nil:
		metadata, err := toAPIDatasetMetadata(*v.DatasetMetadata)
		if err != nil {
			return gatewayapi.ResourceVersionMetadata{}, err
		}
		return gatewayapi.NewDatasetResourceMetadataResourceVersionMetadata(metadata), nil
	case v.SoftwareApplicationMetadata != nil:
		metadata, mapErr := toAPISoftwareApplicationMetadata(*v.SoftwareApplicationMetadata)
		if mapErr != nil {
			return gatewayapi.ResourceVersionMetadata{}, mapErr
		}
		return gatewayapi.NewSoftwareApplicationResourceMetadataResourceVersionMetadata(metadata), nil
	case v.SoftwareSourceCodeMetadata != nil:
		metadata, mapErr := toAPISoftwareSourceCodeMetadata(*v.SoftwareSourceCodeMetadata)
		if mapErr != nil {
			return gatewayapi.ResourceVersionMetadata{}, mapErr
		}
		return gatewayapi.NewSoftwareSourceCodeResourceMetadataResourceVersionMetadata(metadata), nil
	default:
		return gatewayapi.ResourceVersionMetadata{}, errors.New("resource version metadata is nil or unsupported")
	}
}

func toAPIDatasetMetadata(v metadatalifecycleclient.DatasetMetadata) (gatewayapi.DatasetResourceMetadata, error) {
	id, err := parseUUID("dataset.id", v.Id)
	if err != nil {
		return gatewayapi.DatasetResourceMetadata{}, err
	}
	result := gatewayapi.DatasetResourceMetadata{
		ResourceType:       gatewayapi.DatasetResourceMetadataResourceTypeDataset,
		ID:                 id,
		Title:              v.Title,
		Description:        v.Description,
		Keywords:           v.Keywords,
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
		mapped, err := toAPIDistribution(value)
		if err != nil {
			return gatewayapi.DatasetResourceMetadata{}, err
		}
		result.Distributions = append(result.Distributions, mapped)
	}
	return result, nil
}

func toAPISoftwareApplicationMetadata(v metadatalifecycleclient.SoftwareApplicationMetadata) (gatewayapi.SoftwareApplicationResourceMetadata, error) {
	id, err := parseUUID("software_application.id", v.Id)
	if err != nil {
		return gatewayapi.SoftwareApplicationResourceMetadata{}, err
	}
	result := gatewayapi.SoftwareApplicationResourceMetadata{
		ResourceType:     gatewayapi.SoftwareApplicationResourceMetadataResourceTypeSoftwareApplication,
		ID:               id,
		Name:             v.Name,
		Description:      v.Description,
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

func toAPISoftwareSourceCodeMetadata(v metadatalifecycleclient.SoftwareSourceCodeMetadata) (gatewayapi.SoftwareSourceCodeResourceMetadata, error) {
	id, err := parseUUID("software_source_code.id", v.Id)
	if err != nil {
		return gatewayapi.SoftwareSourceCodeResourceMetadata{}, err
	}
	result := gatewayapi.SoftwareSourceCodeResourceMetadata{
		ResourceType:         gatewayapi.SoftwareSourceCodeResourceMetadataResourceTypeSoftwareSourceCode,
		ID:                   id,
		Name:                 v.Name,
		Description:          v.Description,
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

func toAPIReferences(values []metadatalifecycleclient.ResourceReference, field string) ([]gatewayapi.ResourceVersionReference, error) {
	result := make([]gatewayapi.ResourceVersionReference, 0, len(values))
	for _, value := range values {
		id, err := parseUUID(field, value.Id)
		if err != nil {
			return nil, err
		}
		result = append(result, gatewayapi.ResourceVersionReference{
			ID:   id,
			Name: value.Name,
		})
	}
	return result, nil
}

func toAPIIdentifiers(values []metadatalifecycleclient.Identifier) ([]gatewayapi.ResourceIdentifier, error) {
	result := make([]gatewayapi.ResourceIdentifier, 0, len(values))
	for _, value := range values {
		mapped, err := toAPIIdentifier(value)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func toAPIIdentifier(value metadatalifecycleclient.Identifier) (gatewayapi.ResourceIdentifier, error) {
	id, err := parseUUID("resource_identifier.id", value.Id)
	if err != nil {
		return gatewayapi.ResourceIdentifier{}, err
	}
	return gatewayapi.ResourceIdentifier{
		ID:    id,
		Type:  value.Type,
		Value: value.Value,
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
