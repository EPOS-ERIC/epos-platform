package api

import "time"

type Dataset struct {
	BaseDetailResource
	Created     *time.Time `json:"created,omitempty"`
	Modified    *time.Time `json:"modified,omitempty"`
	VersionInfo string     `json:"version_info,omitempty"`
	// TODO: see if we should even return these, even if just returning the ids, they could expose the emails
	ContactPoints      []ResourceRef    `json:"contact_points,omitempty"`
	Keywords           []string         `json:"keywords,omitempty"`
	Concepts           []ResourceRef    `json:"concepts,omitempty"`
	Publisher          []ResourceRef    `json:"publisher,omitempty"`
	Issued             *time.Time       `json:"issued,omitempty"`
	Spatial            string           `json:"spatial,omitempty"`
	Identifiers        []Identifier     `json:"identifiers,omitempty"`
	Temporal           *TemporalExtent  `json:"temporal,omitempty"`
	Status             VersioningStatus `json:"status,omitempty"`
	Distributions      []ResourceRef    `json:"distributions,omitempty"`
	AccessRights       string           `json:"access_rights,omitempty"`
	AccrualPeriodicity string           `json:"accrual_periodicity,omitempty"`
	IsVersionOf        *ResourceRef     `json:"is_version_of,omitempty"`
	QualityAnnotation  string           `json:"quality_annotation,omitempty"`
	Version            string           `json:"version,omitempty"`
}

type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SoftwareApplication struct {
	BaseDetailResource
	ContactPoints        []ResourceRef `json:"contact_points,omitempty"`
	Keywords             []string      `json:"keywords,omitempty"`
	Concepts             []ResourceRef `json:"concepts,omitempty"`
	DownloadURL          string        `json:"download_url,omitempty"`
	InstallURL           string        `json:"install_url,omitempty"`
	License              string        `json:"license,omitempty"`
	SoftwareVersion      string        `json:"software_version,omitempty"`
	SoftwareRequirements string        `json:"software_requirements,omitempty"`
	MainEntityOfPage     string        `json:"main_entity_of_page,omitempty"`
	Provider             []ResourceRef `json:"provider,omitempty"`
	Operations           []ResourceRef `json:"operations,omitempty"`
}

type SoftwareSourceCode struct {
	BaseDetailResource
	ContactPoints       []ResourceRef `json:"contact_points,omitempty"`
	ProgrammingLanguage string        `json:"programming_language,omitempty"`
	Keywords            []string      `json:"keywords,omitempty"`
	Concepts            []ResourceRef `json:"concepts,omitempty"`
	Repository          string        `json:"repository,omitempty"`
	License             string        `json:"license,omitempty"`
	SoftwareVersion     string        `json:"software_version,omitempty"`
	RuntimePlatform     string        `json:"runtime_platform,omitempty"`
	TargetProducts      []ResourceRef `json:"target_products,omitempty"`
	Provider            []ResourceRef `json:"provider,omitempty"`
	MainEntityOfPage    string        `json:"main_entity_of_page,omitempty"`
}

type Distribution struct {
	BaseDetailResource
	DownloadURL string           `json:"download_url,omitempty"`
	Operation   *Operation       `json:"operation,omitempty"`
	Webservice  *ResourceRef     `json:"webservice,omitempty"`
	Format      string           `json:"format,omitempty"`
	Issued      *time.Time       `json:"issued,omitempty"`
	Modified    *time.Time       `json:"modified,omitempty"`
	License     string           `json:"license,omitempty"`
	Type        DistributionKind `json:"type,omitempty"`
}

type Organization struct {
	ID            string        `json:"id"`
	Identifiers   []Identifier  `json:"identifiers,omitempty"`
	Name          string        `json:"name"`
	LeiCode       string        `json:"lei_code,omitempty"`
	Address       string        `json:"address,omitempty"`
	Logo          string        `json:"logo,omitempty"`
	URL           string        `json:"url,omitempty"`
	Email         []string      `json:"email,omitempty"`
	Telephone     []string      `json:"telephone,omitempty"`
	ContactPoints []ResourceRef `json:"contact_points,omitempty"`
	MemberOf      []ResourceRef `json:"member_of,omitempty"`
}

type Webservice struct {
	BaseDetailResource
	Identifiers      []Identifier      `json:"identifiers,omitempty"`
	Entrypoint       string            `json:"entrypoint,omitempty"`
	ContactPoints    []ResourceRef     `json:"contact_points,omitempty"`
	Provider         ResourceRef       `json:"provider"`
	Issued           *time.Time        `json:"issued,omitempty"`
	Modified         *time.Time        `json:"modified,omitempty"`
	Spatial          string            `json:"spatial,omitempty"`
	Temporal         *TemporalExtent   `json:"temporal,omitempty"`
	Keywords         []string          `json:"keywords,omitempty"`
	Concepts         []ResourceRef     `json:"concepts,omitempty"`
	License          string            `json:"license,omitempty"`
	Operations       []Operation       `json:"operations,omitempty"`
	APIDocumentation *APIDocumentation `json:"api_documentation,omitempty"`
}

type Operation struct {
	ID               string            `json:"id"`
	Method           string            `json:"method,omitempty"`
	Returns          []string          `json:"returns,omitempty"`
	IriTemplate      *IriTemplate      `json:"iri_template,omitempty"`
	APIDocumentation *APIDocumentation `json:"api_documentation,omitempty"`
}

type IriTemplate struct {
	Template string               `json:"template"`
	Mappings []IriTemplateMapping `json:"mappings,omitempty"`
}

type IriTemplateMapping struct {
	Variable       string   `json:"variable"`
	Label          string   `json:"label"`
	Range          string   `json:"range"`
	Property       string   `json:"property,omitempty"`
	Required       bool     `json:"required"`
	DefaultValue   string   `json:"default_value,omitempty"`
	MinValue       string   `json:"min_value,omitempty"`
	MaxValue       string   `json:"max_value,omitempty"`
	ParamValue     []string `json:"param_value,omitempty"`
	MultipleValues bool     `json:"multiple_values,omitempty"`
	ReadonlyValue  bool     `json:"readonly_value,omitempty"`
	ValuePattern   string   `json:"value_pattern,omitempty"`
}

type APIDocumentation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Entrypoint  string `json:"entrypoint"`
}

type TemporalExtent struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}
