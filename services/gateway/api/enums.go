package api

import "fmt"

type ResourceType string

const (
	DatasetType             ResourceType = "dataset"
	SoftwareApplicationType ResourceType = "software-application"
	SoftwareSourceCodeType  ResourceType = "software-source-code"
	Organization            ResourceType = "organization"
	WebService              ResourceType = "webservice"
)

func (t ResourceType) IsValid() bool {
	switch t {
	case DatasetType, SoftwareApplicationType, SoftwareSourceCodeType, Organization, WebService:
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
