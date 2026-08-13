package resources

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/resourcehref"
	"github.com/google/uuid"
)

func mapDataset(dataset clients.Dataset) (api.Dataset, error) {
	status, err := api.NewVersioningStatusFromValue(dataset.Status)
	if err != nil && dataset.Status != "" {
		return api.Dataset{}, fmt.Errorf("map dataset status: %w", err)
	}

	return api.Dataset{
		BaseDetailResource: api.BaseDetailResource{
			ID:          dataset.ID.String(),
			Name:        dataset.Title,
			Description: dataset.Description,
		},
		Created:            &dataset.Created,
		Modified:           &dataset.Updated,
		SourceCreated:      dataset.SourceCreated,
		SourceUpdated:      dataset.SourceUpdated,
		Keywords:           splitKeywords(dataset.Keywords),
		Concepts:           categoryRefs(dataset.Categories),
		Publisher:          organizationRefs(dataset.Providers),
		Spatial:            spatialValue(dataset.SpatialCoverage),
		Identifiers:        identifiers(dataset.PersistentIdentifiers),
		Temporal:           temporalValue(dataset.TemporalCoverage),
		Status:             status,
		Distributions:      distributionRefs(dataset.Distributions),
		AccessRights:       stringValue(dataset.AccessRight),
		AccrualPeriodicity: stringValue(dataset.AccrualPeriodicity),
		QualityAnnotation:  stringValue(dataset.QualityAnnotation),
		VersionInfo:        "",
		Issued:             nil,
		IsVersionOf:        nil,
		Version:            "",
	}, nil
}

func mapDistribution(distribution clients.Distribution) (api.Distribution, error) {
	distributionType, err := api.NewDistributionKindFromValue(distribution.Type)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("map distribution type: %w", err)
	}

	result := api.Distribution{
		BaseDetailResource: api.BaseDetailResource{
			ID:          distribution.ID.String(),
			Name:        distribution.Title,
			Description: distribution.Description,
		},
		DownloadURL: "",
		Operation:   nil,
		Webservice:  nil,
		Format:      "",
		Issued:      distribution.Issued,
		Modified:    distribution.Modified,
		License:     stringValue(distribution.License),
		Type:        distributionType,
	}
	if distribution.DownloadableFile != nil {
		result.DownloadURL = distribution.DownloadableFile.URL
		result.Format = distribution.DownloadableFile.Format
	}
	if distribution.WebService != nil {
		result.Webservice = webserviceRef(*distribution.WebService)
		result.Format = distribution.WebService.Format
		operation := mapOperation(*distribution.WebService)
		result.Operation = &operation
	}
	return result, nil
}

func mapWebservice(webservice clients.WebService) api.Webservice {
	return api.Webservice{
		BaseDetailResource: api.BaseDetailResource{
			ID:          webservice.ID.String(),
			Name:        webservice.Name,
			Description: webservice.Description,
		},
		Identifiers:      nil,
		Entrypoint:       webservice.TemplateURL,
		Provider:         organizationRef(webservice.Provider),
		Issued:           nil,
		Modified:         nil,
		Spatial:          spatialValue(webservice.SpatialCoverage),
		Temporal:         temporalValue(webservice.TemporalCoverage),
		Keywords:         nil,
		Concepts:         nil,
		License:          "",
		Operations:       []api.Operation{mapOperation(webservice)},
		APIDocumentation: documentation(webservice.Documentation),
	}
}

func mapSoftwareApplication(software clients.SoftwareApplication) api.SoftwareApplication {
	return api.SoftwareApplication{
		BaseDetailResource: api.BaseDetailResource{
			ID:          software.ID.String(),
			Name:        software.Name,
			Description: software.Description,
		},
		MainEntityOfPage: stringValue(software.MainEntityOfPage),
		License:          stringValue(software.License),
		Version:          stringValue(software.Version),
		Keywords:         splitKeywords(software.Keywords),
		Concepts:         categoryRefs(software.Categories),
		DownloadURL:      stringValue(software.DownloadURL),
		Requirements:     stringValue(software.Requirements),
		Creator:          organizationRefs(software.Creators),
		Identifiers:      identifiers(software.Identifiers),
		OperatingSystem:  stringValue(software.OperatingSystem),
	}
}

func mapSoftwareSourceCode(software clients.SoftwareSourceCode) api.SoftwareSourceCode {
	return api.SoftwareSourceCode{
		BaseDetailResource: api.BaseDetailResource{
			ID:          software.ID.String(),
			Name:        software.Name,
			Description: software.Description,
		},
		ProgrammingLanguage:  stringValue(software.ProgrammingLanguage),
		Keywords:             splitKeywords(software.Keywords),
		Concepts:             categoryRefs(software.Categories),
		Repository:           stringValue(software.Repository),
		License:              stringValue(software.License),
		SoftwareRequirements: stringValue(software.SoftwareRequirements),
		RuntimePlatform:      stringValue(software.RuntimePlatform),
		Creator:              organizationRefs(software.Creators),
		MainEntityOfPage:     stringValue(software.MainEntityOfPage),
		Version:              stringValue(software.Version),
		Identifiers:          identifiers(software.Identifiers),
	}
}

func mapOrganization(organization clients.Organization) api.Organization {
	return api.Organization{
		ID:          organization.ID.String(),
		Identifiers: identifiers(organization.Identifiers),
		Name:        organization.Name,
		LeiCode:     stringValue(organization.LEICode),
		Address: strings.Join(nonEmpty(
			organization.Street,
			organization.Locality,
			organization.PostalCode,
			organization.Country,
		), ", "),
		Logo:          stringValue(organization.LogoURL),
		URL:           stringValue(organization.PageURL),
		Email:         splitSingle(organization.Email),
		Telephone:     nil,
		ContactPoints: nil,
		MemberOf:      nil,
	}
}

func mapConcept(concept clients.Category) api.Concept {
	return api.Concept{
		ID:          concept.ID.String(),
		Name:        concept.Name,
		Description: "",
		InScheme:    categorySchemeRef(concept.Scheme),
		Children:    categoryRefs(concept.Children),
	}
}

func mapConceptScheme(scheme clients.CategoryScheme) api.ConceptScheme {
	return api.ConceptScheme{
		ID:          scheme.ID.String(),
		Label:       stringValue(scheme.Label),
		Name:        scheme.Name,
		Logo:        stringValue(scheme.Logo),
		Homepage:    stringValue(scheme.Page),
		Description: "",
		Color:       stringValue(scheme.Color),
		OrderNumber: scheme.Position,
		TopConcepts: categoryRefs(scheme.TopConcepts),
		Children:    nil,
	}
}

func mapOperation(webservice clients.WebService) api.Operation {
	return api.Operation{
		ID:      webservice.ID.String(),
		Method:  "",
		Returns: nil,
		IriTemplate: &api.IriTemplate{
			Template: webservice.TemplateURL,
			Mappings: parameterMappings(webservice.Parameters),
		},
		APIDocumentation: documentation(webservice.Documentation),
	}
}

func parameterMappings(parameters []clients.Parameter) []api.IriTemplateMapping {
	if len(parameters) == 0 {
		return nil
	}
	mappings := make([]api.IriTemplateMapping, 0, len(parameters))
	for _, parameter := range parameters {
		metadata := parameter.Metadata
		mappings = append(mappings, api.IriTemplateMapping{
			Variable:       metadata.Variable,
			Label:          metadata.Label,
			Range:          metadata.Kind,
			Property:       metadata.Property,
			Required:       metadata.Required,
			DefaultValue:   rawValue(parameter.Default),
			MinValue:       rawValue(parameter.Minimum),
			MaxValue:       rawValue(parameter.Maximum),
			ParamValue:     rawValues(parameter.AllowedValues),
			MultipleValues: metadata.MultipleValues,
			ReadonlyValue:  metadata.ReadOnly,
			ValuePattern:   parameter.Pattern,
		})
	}
	return mappings
}

func rawValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return string(raw)
}

func rawValues(values []json.RawMessage) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, rawValue(value))
	}
	return result
}

func documentation(documentation *clients.Documentation) *api.APIDocumentation {
	if documentation == nil {
		return nil
	}
	return &api.APIDocumentation{
		Title:       stringValue(documentation.Title),
		Description: stringValue(documentation.Description),
		Entrypoint:  documentation.URI,
	}
}

func distributionRefs(distributions []clients.Distribution) []api.ResourceRef {
	result := make([]api.ResourceRef, 0, len(distributions))
	for _, distribution := range distributions {
		result = append(result, api.ResourceRef{
			ID:   distribution.ID.String(),
			Name: distribution.Title,
			Href: resourcehref.Build(api.DistributionType, distribution.ID.String()),
		})
	}
	return result
}

func organizationRefs(organizations []clients.Organization) []api.ResourceRef {
	result := make([]api.ResourceRef, 0, len(organizations))
	for _, organization := range organizations {
		result = append(result, api.ResourceRef{
			ID:   organization.ID.String(),
			Name: organization.Name,
			Href: resourcehref.Build(api.OrganizationType, organization.ID.String()),
		})
	}
	return result
}

func organizationRef(organization *clients.Organization) *api.ResourceRef {
	if organization == nil {
		return nil
	}
	return &api.ResourceRef{
		ID:   organization.ID.String(),
		Name: organization.Name,
		Href: resourcehref.Build(api.OrganizationType, organization.ID.String()),
	}
}

func webserviceRef(webservice clients.WebService) *api.ResourceRef {
	return &api.ResourceRef{
		ID:   webservice.ID.String(),
		Name: webservice.Name,
		Href: resourcehref.Build(api.WebserviceType, webservice.ID.String()),
	}
}

func categoryRefs(categories []clients.Category) []api.ResourceRef {
	result := make([]api.ResourceRef, 0, len(categories))
	for _, category := range categories {
		result = append(result, api.ResourceRef{
			ID:   category.ID.String(),
			Name: category.Name,
			Href: resourcehref.Build(api.ConceptType, category.ID.String()),
		})
	}
	return result
}

func categorySchemeRef(scheme clients.CategoryScheme) *api.ResourceRef {
	if scheme.ID == uuid.Nil {
		return nil
	}
	return &api.ResourceRef{
		ID:   scheme.ID.String(),
		Name: scheme.Name,
		Href: resourcehref.Build(api.ConceptSchemeType, scheme.ID.String()),
	}
}

func identifiers(values []clients.Identifier) []api.Identifier {
	result := make([]api.Identifier, 0, len(values))
	for _, value := range values {
		result = append(result, api.Identifier{
			Type:  value.Type,
			Value: value.Value,
		})
	}
	return result
}

func temporalValue(temporal *clients.Temporal) *api.TemporalExtent {
	if temporal == nil || (temporal.Start == nil && temporal.End == nil) {
		return nil
	}
	return &api.TemporalExtent{
		Start: temporal.Start,
		End:   temporal.End,
	}
}

func spatialValue(spatial *clients.Spatial) string {
	if spatial == nil {
		return ""
	}
	if spatial.Point != nil {
		return fmt.Sprintf("POINT(%g %g)", spatial.Point.Longitude, spatial.Point.Latitude)
	}
	if len(spatial.Polygon) == 0 {
		return ""
	}
	coordinates := make([]string, 0, len(spatial.Polygon))
	for _, coordinate := range spatial.Polygon {
		coordinates = append(coordinates, fmt.Sprintf("%g %g", coordinate.Longitude, coordinate.Latitude))
	}
	return "POLYGON((" + strings.Join(coordinates, ", ") + "))"
}

func splitKeywords(keywords string) []string {
	if keywords == "" {
		return nil
	}

	values := strings.Split(keywords, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}

func splitSingle(value *string) []string {
	if value == nil || *value == "" {
		return nil
	}
	return []string{*value}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nonEmpty(values ...*string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil && *value != "" {
			result = append(result, *value)
		}
	}
	return result
}
