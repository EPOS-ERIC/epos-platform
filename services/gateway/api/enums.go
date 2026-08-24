package api

type ResourceType string

const (
	DatasetType             ResourceType = "dataset"
	SoftwareApplicationType ResourceType = "software-application"
	SoftwareSourceCodeType  ResourceType = "software-source-code"
)

type VersioningStatus string

const (
	PublishedStatus VersioningStatus = "published"
	DraftStatus     VersioningStatus = "draft"
	SubmittedStatus VersioningStatus = "submitted"
	ArchivedStatus  VersioningStatus = "archived"
	DiscardedStatus VersioningStatus = "discarded"
	RejectedStatus  VersioningStatus = "rejected"
)
