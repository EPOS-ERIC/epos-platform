package metadata

type DistributionType string

const (
	DistributionTypeWebService       DistributionType = "http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"
	DistributionTypeDownloadableFile DistributionType = "http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE"
)

// Valid reports whether t is a supported distribution type.
func (t DistributionType) Valid() bool {
	switch t {
	case DistributionTypeDownloadableFile, DistributionTypeWebService:
		return true
	default:
		return false
	}
}

type DatasetStatus string

const (
	DatasetStatusPublished DatasetStatus = "PUBLISHED"
	DatasetStatusDraft     DatasetStatus = "DRAFT"
	DatasetStatusSubmitted DatasetStatus = "SUBMITTED"
	DatasetStatusArchived  DatasetStatus = "ARCHIVED"
	DatasetStatusRejected  DatasetStatus = "REJECTED"
)

// Valid reports whether s is a supported dataset status.
func (s DatasetStatus) Valid() bool {
	switch s {
	case DatasetStatusPublished, DatasetStatusDraft, DatasetStatusSubmitted, DatasetStatusArchived, DatasetStatusRejected:
		return true
	default:
		return false
	}
}

type ParameterKind string

const (
	ParameterKindString   ParameterKind = "STRING"
	ParameterKindBoolean  ParameterKind = "BOOLEAN"
	ParameterKindInteger  ParameterKind = "INTEGER"
	ParameterKindFloat    ParameterKind = "FLOAT"
	ParameterKindDateTime ParameterKind = "DATETIME"
)

// Valid reports whether k is a supported parameter kind.
func (k ParameterKind) Valid() bool {
	switch k {
	case ParameterKindString, ParameterKindBoolean, ParameterKindInteger, ParameterKindFloat, ParameterKindDateTime:
		return true
	default:
		return false
	}
}

type IdentifierType string

type ContactPointRole string

type TopLevelConcept string

const (
	TopLevelConceptData       TopLevelConcept = "DATA"
	TopLevelConceptFacilities TopLevelConcept = "FACILITIES"
	TopLevelConceptSoftware   TopLevelConcept = "SOFTWARE"
)

// Valid reports whether c is a supported top-level concept.
func (c TopLevelConcept) Valid() bool {
	switch c {
	case TopLevelConceptData, TopLevelConceptFacilities, TopLevelConceptSoftware:
		return true
	default:
		return false
	}
}
