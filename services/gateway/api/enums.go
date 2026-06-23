package api

import (
	"fmt"
	"strings"
)

type ResourceType string

const (
	DatasetType             ResourceType = "dataset"
	SoftwareApplicationType ResourceType = "softwareapplication"
	SoftwareSourceCodeType  ResourceType = "softwaresourcecode"
	OrganizationType        ResourceType = "organization"
	WebserviceType          ResourceType = "webservice"
	DistributionType        ResourceType = "distribution"
	ConceptType             ResourceType = "concept"
	ConceptSchemeType       ResourceType = "conceptscheme"
)

func (t ResourceType) IsValid() bool {
	switch t {
	case DatasetType, SoftwareApplicationType, SoftwareSourceCodeType, OrganizationType, WebserviceType, DistributionType, ConceptType, ConceptSchemeType:
		return true
	default:
		return false
	}
}

func NewResourceTypeFromValue(value string) (ResourceType, error) {
	resourceType := ResourceType(value)
	if resourceType.IsValid() {
		return resourceType, nil
	}

	return "", fmt.Errorf("invalid value '%s' for ResourceType", value)
}

type VersioningStatus string

const (
	PublishedStatus VersioningStatus = "published"
	DraftStatus     VersioningStatus = "draft"
	SubmittedStatus VersioningStatus = "submitted"
	ArchivedStatus  VersioningStatus = "archived"
	DiscardedStatus VersioningStatus = "discarded"
)

func (s VersioningStatus) IsValid() bool {
	switch s {
	case PublishedStatus, DraftStatus, SubmittedStatus, ArchivedStatus, DiscardedStatus:
		return true
	default:
		return false
	}
}

func NewVersioningStatusFromValue(value string) (VersioningStatus, error) {
	status := VersioningStatus(strings.ToLower(strings.TrimSpace(value)))
	if status.IsValid() {
		return status, nil
	}

	return "", fmt.Errorf("invalid value '%s' for VersioningStatus", value)
}

type DistributionKind string

const (
	DistributionKindWebService       DistributionKind = "http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"
	DistributionKindDownloadableFile DistributionKind = "http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE"
)

func (t DistributionKind) IsValid() bool {
	switch t {
	case DistributionKindWebService, DistributionKindDownloadableFile:
		return true
	default:
		return false
	}
}

func NewDistributionKindFromValue(value string) (DistributionKind, error) {
	distributionKind := DistributionKind(value)
	if distributionKind.IsValid() {
		return distributionKind, nil
	}

	return "", fmt.Errorf("invalid value '%s' for DistributionKind", value)
}
