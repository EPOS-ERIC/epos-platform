package api

import "fmt"

type ResourceType string

const (
	DatasetType             ResourceType = "dataset"
	SoftwareApplicationType ResourceType = "softwareapplication"
	SoftwareSourceCodeType  ResourceType = "softwaresourcecode"
	OrganizationType        ResourceType = "organization"
	WebserviceType          ResourceType = "webservice"
	DistributionType        ResourceType = "distribution"
	ConceptType             ResourceType = "concept"
)

func (t ResourceType) IsValid() bool {
	switch t {
	case DatasetType, SoftwareApplicationType, SoftwareSourceCodeType, OrganizationType, WebserviceType, DistributionType, ConceptType:
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

	return "", fmt.Errorf("invalid value %q for ResourceType", value)
}

type VersioningStatus string

const (
	DraftStatus     VersioningStatus = "DRAFT"
	SubmittedStatus VersioningStatus = "SUBMITTED"
	PublishedStatus VersioningStatus = "PUBLISHED"
	ArchivedStatus  VersioningStatus = "ARCHIVED"
)

func (s VersioningStatus) IsValid() bool {
	switch s {
	case DraftStatus, SubmittedStatus, PublishedStatus, ArchivedStatus:
		return true
	default:
		return false
	}
}

func NewVersioningStatusFromValue(value string) (VersioningStatus, error) {
	status := VersioningStatus(value)
	if status.IsValid() {
		return status, nil
	}

	return "", fmt.Errorf("invalid value %q for VersioningStatus", value)
}
