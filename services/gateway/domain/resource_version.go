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
	IsDataset                 bool
	ConversionsByDistribution map[uuid.UUID][]AvailableConversion
}

type Relation struct {
	ID             uuid.UUID
	DistributionID uuid.UUID
	PluginID       uuid.UUID
	InputFormat    string
	OutputFormat   string
	CreatedBy      string
	UpdatedBy      string
}

// Equivalent reports whether relations have the same distribution, plugin, and formats.
func (r Relation) Equivalent(other Relation) bool {
	return r.DistributionID == other.DistributionID &&
		r.PluginID == other.PluginID &&
		r.InputFormat == other.InputFormat &&
		r.OutputFormat == other.OutputFormat
}
