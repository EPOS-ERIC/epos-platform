package domain

import "github.com/google/uuid"

type ResourceType string

const (
	ResourceTypeDataset             ResourceType = "dataset"
	ResourceTypeSoftwareApplication ResourceType = "software-application"
	ResourceTypeSoftwareSourceCode  ResourceType = "software-source-code"
)

// ResourceVersion contains the fields needed to enrich a resource response.
type ResourceVersion struct {
	IsDataset     bool
	Distributions []ResourceDistribution
}

type ResourceDistribution struct {
	ID          uuid.UUID
	Conversions []AvailableConversion
}
