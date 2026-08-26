// Package turtle parses Turtle metadata into domain import batches.
package turtle

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	domain "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadataimport"
	rdf "github.com/deiu/rdf2go"
	"github.com/google/uuid"
)

type (
	Options          = metadataimport.Options
	Batch            = metadataimport.Batch
	CategoryEdge     = metadataimport.CategoryEdge
	OrganizationEdge = metadataimport.OrganizationEdge
	Severity         = metadataimport.Severity
	Diagnostic       = metadataimport.Diagnostic
)

const (
	SeverityWarning = metadataimport.SeverityWarning
	SeverityError   = metadataimport.SeverityError
)

type Parser struct{}

// Parse converts Turtle input into a metadata import batch.
func (Parser) Parse(input io.Reader, options metadataimport.Options) (metadataimport.Batch, error) {
	return Parse(input, options)
}

// Parse converts Turtle input into a metadata import batch.
func Parse(input io.Reader, options Options) (Batch, error) {
	if options.BaseURI == "" {
		options.BaseURI = "https://epos-eu.org/metadata-lifecycle/import"
	}

	g, err := parseGraph(input, options.BaseURI)
	if err != nil {
		return Batch{}, err
	}

	return mapGraph(g, options)
}

type mapper struct {
	graph             *graph
	options           Options
	warnings          []Diagnostic
	schemes           map[string]domain.CategoryScheme
	categories        map[string]domain.Category
	organizations     map[string]domain.Organization
	contacts          map[string]domain.ContactPoint
	categoryEdges     map[string]CategoryEdge
	organizationEdges map[string]OrganizationEdge
}

func mapGraph(g *graph, options Options) (Batch, error) {
	m := &mapper{
		graph:             g,
		options:           options,
		warnings:          nil,
		schemes:           make(map[string]domain.CategoryScheme),
		categories:        make(map[string]domain.Category),
		organizations:     make(map[string]domain.Organization),
		contacts:          make(map[string]domain.ContactPoint),
		categoryEdges:     make(map[string]CategoryEdge),
		organizationEdges: make(map[string]OrganizationEdge),
	}

	var datasets []domain.Dataset
	var softwareApplications []domain.SoftwareApplication
	var softwareSourceCodes []domain.SoftwareSourceCode

	for _, source := range m.graph.subjects(rdfType, dcatDataset) {
		dataset, err := m.dataset(source)
		if err != nil {
			return Batch{}, err
		}
		datasets = append(datasets, dataset)
	}

	for _, source := range m.graph.subjects(rdfType, schemaSoftwareApplication) {
		software, err := m.softwareApplication(source)
		if err != nil {
			return Batch{}, err
		}
		softwareApplications = append(softwareApplications, software)
	}

	for _, source := range m.graph.subjects(rdfType, schemaSoftwareSourceCode) {
		software, err := m.softwareSourceCode(source)
		if err != nil {
			return Batch{}, err
		}
		softwareSourceCodes = append(softwareSourceCodes, software)
	}

	for _, source := range m.graph.subjects(rdfType, skosConceptScheme) {
		if !m.supportedScheme(source) {
			continue
		}
		if _, err := m.scheme(source); err != nil {
			return Batch{}, err
		}
	}
	for _, source := range m.graph.subjects(rdfType, skosConcept) {
		if _, ok := m.firstString(source, skosInScheme); !ok {
			continue
		}
		schemeIRI, _ := m.firstString(source, skosInScheme)
		if !m.supportedScheme(schemeIRI) {
			continue
		}
		if _, err := m.category(source, make(map[string]bool)); err != nil {
			return Batch{}, err
		}
	}
	for _, source := range m.graph.subjects(rdfType, schemaOrganization) {
		if _, err := m.organization(source, make(map[string]bool)); err != nil {
			return Batch{}, err
		}
	}
	for _, source := range m.graph.subjects(rdfType, schemaContactPoint) {
		if _, err := m.contact(source); err != nil {
			return Batch{}, err
		}
	}

	m.addUnsupportedWarnings()

	if len(datasets) == 0 &&
		len(softwareApplications) == 0 &&
		len(softwareSourceCodes) == 0 &&
		len(m.schemes) == 0 &&
		len(m.categories) == 0 &&
		len(m.organizations) == 0 &&
		len(m.contacts) == 0 {
		return Batch{}, fmt.Errorf("no supported metadata resources found")
	}

	for _, dataset := range datasets {
		if err := dataset.Validate(); err != nil {
			return Batch{}, fmt.Errorf("validate dataset %q: %w", dataset.SourceIRI, err)
		}
	}

	for _, software := range softwareApplications {
		if err := software.Validate(); err != nil {
			return Batch{}, fmt.Errorf("validate software application %q: %w", software.SourceIRI, err)
		}
	}

	for _, software := range softwareSourceCodes {
		if err := software.Validate(); err != nil {
			return Batch{}, fmt.Errorf("validate software source code %q: %w", software.SourceIRI, err)
		}
	}

	return Batch{
		CategorySchemes:      mapValues(m.schemes),
		Categories:           mapValues(m.categories),
		Organizations:        mapValues(m.organizations),
		ContactPoints:        mapValues(m.contacts),
		Datasets:             datasets,
		SoftwareApplications: softwareApplications,
		SoftwareSourceCodes:  softwareSourceCodes,
		CategoryEdges:        mapEdges(m.categoryEdges),
		OrganizationEdges:    mapOrganizationEdges(m.organizationEdges),
		Warnings:             m.warnings,
	}, nil
}

func (m *mapper) softwareApplication(source string) (domain.SoftwareApplication, error) {
	result := domain.SoftwareApplication{
		ID:               sharedID("software-application", source),
		SourceIRI:        source,
		Name:             m.requiredString(source, schemaName),
		Description:      m.optionalString(source, schemaDescription),
		MainEntityOfPage: stringPtr(m.optionalString(source, schemaMainEntityOfPage)),
		License:          stringPtr(m.optionalString(source, schemaLicense)),
		Version:          stringPtr(m.optionalStringOneOf(source, schemaSoftwareVersion, schemaVersion)),
		Keywords:         strings.Join(m.strings(source, schemaKeywords), ","),
		DownloadURL:      stringPtr(m.optionalString(source, schemaDownloadURL)),
		Requirements:     stringPtr(m.optionalString(source, schemaSoftwareRequirements)),
		OperatingSystem:  stringPtr(m.optionalString(source, schemaOperatingSystem)),
		Creators:         nil,
		Identifiers:      nil,
		ContactPoints:    nil,
		Categories:       nil,
	}

	if err := m.addSoftwareRelations(source, result.ID, &result.Creators, &result.Identifiers, &result.ContactPoints, &result.Categories); err != nil {
		return domain.SoftwareApplication{}, err
	}

	return result, nil
}

func (m *mapper) softwareSourceCode(source string) (domain.SoftwareSourceCode, error) {
	result := domain.SoftwareSourceCode{
		ID:                   sharedID("software-source-code", source),
		SourceIRI:            source,
		Name:                 m.requiredString(source, schemaName),
		Description:          m.optionalString(source, schemaDescription),
		ProgrammingLanguage:  stringPtr(m.optionalString(source, schemaProgrammingLanguage)),
		Keywords:             strings.Join(m.strings(source, schemaKeywords), ","),
		Repository:           stringPtr(m.optionalString(source, schemaCodeRepository)),
		License:              stringPtr(m.optionalString(source, schemaLicense)),
		SoftwareRequirements: stringPtr(m.optionalString(source, schemaSoftwareRequirements)),
		RuntimePlatform:      stringPtr(m.optionalString(source, schemaRuntimePlatform)),
		MainEntityOfPage:     stringPtr(m.optionalString(source, schemaMainEntityOfPage)),
		Version:              stringPtr(m.optionalString(source, schemaVersion)),
		Creators:             nil,
		Identifiers:          nil,
		ContactPoints:        nil,
		Categories:           nil,
	}

	if err := m.addSoftwareRelations(source, result.ID, &result.Creators, &result.Identifiers, &result.ContactPoints, &result.Categories); err != nil {
		return domain.SoftwareSourceCode{}, err
	}

	return result, nil
}

func (m *mapper) addSoftwareRelations(
	source string,
	owner uuid.UUID,
	creators *[]domain.Organization,
	identifiers *[]domain.Identifier,
	contacts *[]domain.ContactPoint,
	categories *[]domain.Category,
) error {
	for _, identifierSource := range m.graph.refs(source, schemaIdentifier) {
		if identifier, ok := m.identifier(identifierSource, owner, source); ok {
			*identifiers = append(*identifiers, identifier)
		}
	}

	for _, creatorSource := range m.graph.refs(source, schemaCreator) {
		creator, err := m.organization(creatorSource, make(map[string]bool))
		if err != nil {
			return err
		}
		*creators = append(*creators, creator)
	}

	for _, contactSource := range m.graph.refs(source, dcatContactPoint) {
		contact, err := m.contact(contactSource)
		if err != nil {
			return err
		}
		*contacts = append(*contacts, contact)
	}

	for _, categorySource := range m.graph.refs(source, dcatTheme) {
		category, err := m.category(categorySource, map[string]bool{})
		if err != nil {
			return err
		}
		*categories = append(*categories, category)
	}

	return nil
}

func (m *mapper) dataset(source string) (domain.Dataset, error) {
	id := ownedID("", "dataset", source)
	dataset := domain.Dataset{
		ID:                    id,
		SourceIRI:             source,
		Title:                 "",
		Description:           "",
		Keywords:              "",
		AccrualPeriodicity:    nil,
		QualityAnnotation:     nil,
		AccessRight:           nil,
		Distributions:         nil,
		SpatialCoverage:       nil,
		TemporalCoverage:      nil,
		PersistentIdentifiers: nil,
		ContactPoints:         nil,
		Providers:             nil,
		Categories:            nil,
		Status:                domain.DatasetStatusDraft,
		Created:               time.Time{},
		Updated:               time.Time{},
		SourceCreated:         nil,
		SourceUpdated:         nil,
	}
	dataset.Title = m.requiredString(source, dctTitle)
	dataset.Description = m.optionalString(source, dctDescription)
	dataset.Keywords = strings.Join(m.strings(source, dcatKeyword), ",")

	if value := m.optionalString(source, dctAccrualPeriodicity); value != "" {
		dataset.AccrualPeriodicity = &value
	}

	if annotation, ok := m.firstString(source, dqvHasQualityAnnotation); ok {
		if value := m.optionalString(annotation, oaHasBody); value != "" {
			dataset.QualityAnnotation = &value
		}
	}

	if value := m.optionalString(source, dctAccessRights); value != "" {
		dataset.AccessRight = &value
	}

	if created := m.date(source, dctCreated); !created.IsZero() {
		dataset.SourceCreated = &created
	}

	if updated := m.date(source, dctModified); !updated.IsZero() {
		dataset.SourceUpdated = &updated
	}

	dataset.PersistentIdentifiers = append(dataset.PersistentIdentifiers, m.identifiers(source, id)...)

	for _, identifierSource := range m.graph.refs(source, admsIdentifier) {
		identifier, ok := m.identifier(identifierSource, id, source)
		if ok {
			dataset.PersistentIdentifiers = append(dataset.PersistentIdentifiers, identifier)
		}
	}

	if value, ok := m.first(source, dctTemporal); ok {
		dataset.TemporalCoverage = m.temporal(value.RawValue())
	}

	if value, ok := m.first(source, dctSpatial); ok {
		dataset.SpatialCoverage = m.spatial(value.RawValue())
	}

	for _, ref := range m.graph.refs(source, dcatTheme) {
		category, err := m.category(ref, map[string]bool{})
		if err != nil {
			return domain.Dataset{}, err
		}
		dataset.Categories = append(dataset.Categories, category)
	}

	for _, ref := range m.graph.refs(source, dcatContactPoint) {
		contact, err := m.contact(ref)
		if err != nil {
			return domain.Dataset{}, err
		}
		dataset.ContactPoints = append(dataset.ContactPoints, contact)
	}

	for _, ref := range m.graph.refs(source, dctPublisher) {
		organization, err := m.organization(ref, make(map[string]bool))
		if err != nil {
			return domain.Dataset{}, err
		}
		dataset.Providers = append(dataset.Providers, organization)
	}

	for _, ref := range m.graph.refs(source, dcatDistributionLink) {
		distribution, err := m.distribution(ref, id)
		if err != nil {
			return domain.Dataset{}, err
		}
		dataset.Distributions = append(dataset.Distributions, distribution)
	}

	return dataset, nil
}

func (m *mapper) distribution(source string, owner uuid.UUID) (domain.Distribution, error) {
	distribution := domain.Distribution{
		ID:               ownedID(owner.String(), "distribution", source),
		Title:            m.requiredString(source, dctTitle),
		Description:      m.optionalString(source, dctDescription),
		License:          nil,
		Type:             "",
		Issued:           nil,
		Modified:         nil,
		ByteSize:         m.intPtr(source, dcatByteSize),
		WebService:       nil,
		DownloadableFile: nil,
	}

	if license, ok := m.firstString(source, dctLicense); ok {
		distribution.License = &license
	}

	if issued := m.date(source, dctIssued); !issued.IsZero() {
		distribution.Issued = &issued
	}

	if modified := m.date(source, dctModified); !modified.IsZero() {
		distribution.Modified = &modified
	}

	if service, ok := m.firstString(source, dctConformsTo); ok && m.graph.hasType(service, "https://www.epos-eu.org/epos-dcat-ap#WebService") {
		webService, err := m.webService(service, distribution.ID, m.optionalString(source, dctFormat))
		if err != nil {
			return domain.Distribution{}, err
		}
		distribution.WebService = &webService
	} else if url, ok := m.firstString(source, dcatDownloadURL); ok {
		format := m.optionalString(source, dctFormat)
		distribution.DownloadableFile = &domain.DownloadableFile{URL: url, Format: format}
	} else if url, ok := m.firstString(source, dcatAccessURL); ok {
		format := m.optionalString(source, dctFormat)
		distribution.DownloadableFile = &domain.DownloadableFile{URL: url, Format: format}
	}

	typeValue, err := distribution.DeliveryType()
	if err != nil {
		return domain.Distribution{}, fmt.Errorf("distribution %q: %w", source, err)
	}

	distribution.Type = typeValue

	return distribution, nil
}

func (m *mapper) webService(source string, owner uuid.UUID, format string) (domain.WebService, error) {
	service := domain.WebService{
		ID:               ownedID(owner.String(), "web-service", source),
		Name:             m.optionalString(source, schemaName),
		Description:      m.optionalString(source, schemaDescription),
		TemplateURL:      "",
		Parameters:       nil,
		Format:           format,
		Provider:         nil,
		Documentation:    nil,
		SpatialCoverage:  nil,
		TemporalCoverage: nil,
		ContactPoints:    nil,
	}

	if service.Name == "" {
		service.Name = m.optionalString(source, dctTitle)
	}

	if value, ok := m.first(source, dctTemporal); ok {
		service.TemporalCoverage = m.temporal(value.RawValue())
	}

	if value, ok := m.first(source, dctSpatial); ok {
		service.SpatialCoverage = m.spatial(value.RawValue())
	}

	providers := m.graph.refs(source, schemaProvider)
	if len(providers) > 1 {
		return domain.WebService{}, fmt.Errorf("web service must have at most one provider")
	}
	if len(providers) == 1 {
		item, err := m.organization(providers[0], make(map[string]bool))
		if err != nil {
			return domain.WebService{}, err
		}
		service.Provider = &item
	}

	for _, contact := range m.graph.refs(source, dcatContactPoint) {
		item, err := m.contact(contact)
		if err != nil {
			return domain.WebService{}, err
		}
		service.ContactPoints = append(service.ContactPoints, item)
	}

	for _, documentation := range m.graph.refs(source, dctConformsTo) {
		if m.graph.hasType(documentation, hydraAPIDocumentation) {
			item := domain.Documentation{
				Title:       stringPtr(m.optionalString(documentation, hydraTitle)),
				Description: stringPtr(m.optionalString(documentation, hydraDescription)),
				URI:         m.optionalString(documentation, hydraEntrypoint),
			}
			if item.URI == "" {
				item.URI = documentation
			}
			service.Documentation = &item
		}
	}

	for _, operation := range m.graph.refs(source, hydraSupportedOperation) {
		if template, ok := m.firstString(operation, hydraTemplate); ok && service.TemplateURL == "" {
			service.TemplateURL = template
		}
		for _, property := range m.graph.refs(operation, hydraProperty) {
			if template, ok := m.firstString(property, hydraTemplate); ok && service.TemplateURL == "" {
				service.TemplateURL = template
			}
			for _, mapping := range m.graph.refs(property, hydraMapping) {
				parameter, err := m.parameter(mapping, service.ID)
				if err != nil {
					return domain.WebService{}, err
				}
				service.Parameters = append(service.Parameters, parameter)
			}
		}
	}

	return service, nil
}

func (m *mapper) parameter(source string, owner uuid.UUID) (domain.Parameter, error) {
	metadata := domain.ParameterMetadata{
		ID:             ownedID(owner.String(), "parameter", source),
		Kind:           "",
		Variable:       m.requiredString(source, hydraVariable),
		Label:          m.optionalString(source, "http://www.w3.org/2000/01/rdf-schema#label"),
		Property:       m.optionalString(source, hydraProperty),
		Required:       m.bool(source, hydraRequired),
		MultipleValues: m.bool(source, schemaMultipleValues),
		ReadOnly:       m.bool(source, schemaReadonlyValue),
	}
	if metadata.Variable == "" {
		return nil, fmt.Errorf("parameter %q has no variable", source)
	}
	if metadata.Label == "" {
		metadata.Label = metadata.Variable
	}
	if metadata.Property == "" {
		metadata.Property = metadata.Variable
	}
	rangeIRI := normalizeRange(m.optionalString(source, rdfsRange))
	config := func() (string, []string) {
		return m.optionalString(source, schemaDefaultValue), m.strings(source, httpParamValue)
	}
	switch rangeIRI {
	case rdfsDatatypeBoolean:
		defaultText, hasDefault := m.firstString(source, schemaDefaultValue)
		defaultValueText := strings.EqualFold(defaultText, "true")
		var defaultValue *bool
		if hasDefault {
			defaultValue = &defaultValueText
		}
		return domain.BooleanParameter{
			Metadata: metadata,
			Default:  defaultValue,
		}, nil
	case rdfsDatatypeInteger:
		defaultValue := m.intPtr(source, schemaDefaultValue)
		return domain.IntegerParameter{
			Metadata:      metadata,
			Minimum:       m.intPtr(source, schemaMinValue),
			Maximum:       m.intPtr(source, schemaMaxValue),
			Default:       defaultValue,
			AllowedValues: m.validIntValues(source, defaultValue),
		}, nil
	case rdfsDatatypeFloat:
		defaultValue := m.floatPtr(source, schemaDefaultValue)
		return domain.FloatParameter{
			Metadata:      metadata,
			Minimum:       m.floatPtr(source, schemaMinValue),
			Maximum:       m.floatPtr(source, schemaMaxValue),
			Default:       defaultValue,
			AllowedValues: m.validFloatValues(source, defaultValue),
		}, nil
	case rdfsDatatypeDateTime:
		return domain.DateTimeParameter{
			Metadata: metadata,
			Minimum:  timePtr(m.time(source, schemaMinValue)),
			Maximum:  timePtr(m.time(source, schemaMaxValue)),
			Default:  timePtr(m.time(source, schemaDefaultValue)),
			Pattern:  m.optionalString(source, schemaValuePattern),
		}, nil
	default:
		defaultValue, allowed := config()
		var pointer *string
		if defaultValue != "" {
			pointer = &defaultValue
		}
		if pointer != nil && len(allowed) > 0 && !containsString(allowed, *pointer) {
			m.warnings = append(m.warnings, Diagnostic{
				Severity:  SeverityWarning,
				Subject:   m.warningSubject(source),
				Predicate: "",
				Message:   "default value is not in paramValue list; allowed values ignored",
			})
			allowed = nil
		}
		return domain.StringParameter{
			Metadata:      metadata,
			Default:       pointer,
			AllowedValues: allowed,
			Pattern:       m.optionalString(source, schemaValuePattern),
		}, nil
	}
}

func (m *mapper) scheme(source string) (domain.CategoryScheme, error) {
	if value, ok := m.schemes[source]; ok {
		return value, nil
	}
	concept := domain.TopLevelConcept("")
	for _, top := range m.graph.refs(source, skosHasTopConcept) {
		switch top {
		case "category:facets/dataset-theme":
			concept = domain.TopLevelConceptData
		case "category:facets/facility-theme":
			concept = domain.TopLevelConceptFacilities
		case "category:facets/software-theme":
			concept = domain.TopLevelConceptSoftware
		}
	}
	result := domain.CategoryScheme{
		ID:          sharedID("category-scheme", source),
		SourceIRI:   source,
		Name:        m.optionalString(source, dctTitle),
		Description: stringPtr(m.optionalString(source, dctDescription)),
		Label:       stringPtr(m.optionalString(source, skosPrefLabel)),
		Logo:        stringPtr(m.optionalString(source, foafLogo)),
		Page:        stringPtr(m.optionalString(source, foafHomepage)),
		Color:       stringPtr(m.optionalString(source, schemaColor)),
		Position:    int32(m.int64(source, schemaOrderItemNumber)),
		Concept:     domain.TopLevelConcept(""),
	}
	if result.Name == "" {
		result.Name = m.optionalString(source, skosPrefLabel)
	}
	result.Concept = concept
	if !result.Concept.Valid() {
		return domain.CategoryScheme{}, fmt.Errorf("category scheme %q has unsupported top concept", source)
	}
	m.schemes[source] = result
	return result, nil
}

func (m *mapper) supportedScheme(source string) bool {
	for _, top := range m.graph.refs(source, skosHasTopConcept) {
		if top == "category:facets/dataset-theme" || top == "category:facets/facility-theme" || top == "category:facets/software-theme" {
			return true
		}
	}
	return false
}

func (m *mapper) category(source string, active map[string]bool) (domain.Category, error) {
	if active[source] {
		return domain.Category{}, fmt.Errorf("category hierarchy cycle at %q", source)
	}
	if value, ok := m.categories[source]; ok {
		return value, nil
	}
	active[source] = true
	defer delete(active, source)
	schemeIRI, ok := m.firstString(source, skosInScheme)
	if !ok {
		return domain.Category{}, fmt.Errorf("category %q has no scheme", source)
	}
	scheme, err := m.scheme(schemeIRI)
	if err != nil {
		return domain.Category{}, err
	}
	result := domain.Category{
		ID:        sharedID("category", source),
		SourceIRI: source,
		Name:      m.requiredString(source, skosPrefLabel),
		Scheme:    scheme,
		Children:  nil,
	}
	parents := m.graph.refs(source, skosBroader)
	if len(parents) > 1 {
		return domain.Category{}, fmt.Errorf("category %q has multiple parents", source)
	}
	for _, parentIRI := range parents {
		parent, err := m.category(parentIRI, active)
		if err != nil {
			return domain.Category{}, err
		}
		m.categoryEdges[parent.SourceIRI+"\x00"+result.SourceIRI] = CategoryEdge{ParentID: parent.ID, ChildID: result.ID}
	}
	m.categories[source] = result
	return result, nil
}

func (m *mapper) organization(source string, active map[string]bool) (domain.Organization, error) {
	if active[source] {
		return domain.Organization{}, fmt.Errorf("organization hierarchy cycle at %q", source)
	}
	if value, ok := m.organizations[source]; ok {
		return value, nil
	}
	active[source] = true
	defer delete(active, source)
	address := source
	if references := m.graph.refs(source, schemaAddress); len(references) > 0 {
		address = references[0]
	}
	result := domain.Organization{
		ID:            sharedID("organization", source),
		SourceIRI:     source,
		Name:          m.optionalString(source, schemaLegalName),
		Street:        stringPtr(m.optionalString(address, schemaStreetAddress)),
		Locality:      stringPtr(m.optionalString(address, schemaAddressLocality)),
		PostalCode:    stringPtr(m.optionalString(address, schemaPostalCode)),
		Country:       stringPtr(m.optionalString(address, schemaAddressCountry)),
		LogoURL:       stringPtr(m.optionalString(source, schemaLogo)),
		PageURL:       stringPtr(m.optionalString(source, schemaURL)),
		Email:         stringPtr(m.optionalString(source, schemaEmail)),
		LEICode:       stringPtr(m.optionalString(source, schemaLEICode)),
		ContactPoints: nil,
		Identifiers:   nil,
	}
	if result.Name == "" {
		result.Name = m.optionalString(source, schemaName)
	}
	if result.Name == "" {
		return domain.Organization{}, fmt.Errorf("organization %q has no name", source)
	}
	for _, identifierSource := range m.graph.refs(source, schemaIdentifier) {
		if identifier, ok := m.identifier(identifierSource, result.ID, source); ok {
			result.Identifiers = append(result.Identifiers, identifier)
		}
	}
	for _, contactSource := range m.graph.refs(source, dcatContactPoint) {
		contact, err := m.contact(contactSource)
		if err != nil {
			return domain.Organization{}, err
		}
		result.ContactPoints = append(result.ContactPoints, contact)
	}
	for _, parentIRI := range m.graph.refs(source, schemaMemberOf) {
		parent, err := m.organization(parentIRI, active)
		if err != nil {
			return domain.Organization{}, err
		}
		m.organizationEdges[parent.SourceIRI+"\x00"+result.SourceIRI] = OrganizationEdge{ParentID: parent.ID, ChildID: result.ID}
	}
	m.organizations[source] = result
	return result, nil
}

func normalizeRange(value string) string {
	switch value {
	case "xsd:boolean":
		return rdfsDatatypeBoolean
	case "xsd:integer":
		return rdfsDatatypeInteger
	case "xsd:float":
		return rdfsDatatypeFloat
	case "xsd:dateTime":
		return rdfsDatatypeDateTime
	default:
		return value
	}
}

func (m *mapper) contact(source string) (domain.ContactPoint, error) {
	if value, ok := m.contacts[source]; ok {
		return value, nil
	}
	email := m.optionalString(source, schemaEmail)
	role := m.optionalString(source, schemaContactType)
	if email == "" || role == "" {
		return domain.ContactPoint{}, fmt.Errorf("contact point %q requires email and role", source)
	}
	result := domain.ContactPoint{
		ID:        sharedID("contact-point", source),
		SourceIRI: source,
		Email:     email,
		Role:      domain.ContactPointRole(role),
	}
	m.contacts[source] = result
	return result, nil
}

func (m *mapper) identifiers(subject string, owner uuid.UUID) []domain.Identifier {
	result := make([]domain.Identifier, 0)
	for _, identifierSource := range m.graph.refs(subject, dctIdentifier) {
		if identifier, ok := m.identifier(identifierSource, owner, subject); ok {
			result = append(result, identifier)
		}
	}
	return result
}

func (m *mapper) identifier(source string, owner uuid.UUID, ownerIRI string) (domain.Identifier, bool) {
	typ := m.optionalString(source, admsSchemeAgency)
	value := m.optionalString(source, "http://www.w3.org/2004/02/skos/core#notation")
	if typ == "" {
		typ = m.optionalString(source, "http://schema.org/propertyID")
	}
	if value == "" {
		value = m.optionalString(source, "http://schema.org/value")
	}
	if value == "" {
		value = m.optionalString(source, skosPrefLabel)
	}
	if typ == "" || value == "" {
		return domain.Identifier{ID: uuid.Nil, SourceIRI: "", Type: "", Value: ""}, false
	}
	key := source
	sourceIRI := source
	if ownerIRI != "" {
		key = ownerIRI + "\x00" + typ + "\x00" + value
		sourceIRI = source + "|owner=" + ownerIRI + "|type=" + typ + "|value=" + value
	}
	return domain.Identifier{
		ID:        ownedID(owner.String(), "identifier", key),
		SourceIRI: sourceIRI,
		Type:      domain.IdentifierType(typ),
		Value:     value,
	}, true
}

func (m *mapper) addUnsupportedWarnings() {
	unsupported := map[string]bool{
		"http://schema.org/Person":                       true,
		"https://www.epos-eu.org/epos-dcat-ap#Facility":  true,
		"https://www.epos-eu.org/epos-dcat-ap#Equipment": true,
	}
	for subject, predicates := range m.graph.values {
		for _, value := range predicates[rdfType] {
			if unsupported[value.RawValue()] {
				m.warnings = append(m.warnings, Diagnostic{
					Severity:  SeverityWarning,
					Subject:   subject,
					Predicate: "",
					Message:   "unsupported RDF type ignored: " + value.RawValue(),
				})
			}
		}
	}
}

func (m *mapper) first(subject, predicate string) (rdf.Term, bool) {
	return m.graph.first(subject, predicate)
}

func (m *mapper) firstString(subject, predicate string) (string, bool) {
	return m.graph.firstString(subject, predicate)
}

func (m *mapper) requiredString(subject, predicate string) string {
	return m.optionalString(subject, predicate)
}

func (m *mapper) optionalString(subject, predicate string) string {
	value, _ := m.graph.firstString(subject, predicate)
	return value
}

func (m *mapper) optionalStringOneOf(subject string, predicates ...string) string {
	for _, predicate := range predicates {
		if value := m.optionalString(subject, predicate); value != "" {
			return value
		}
	}
	return ""
}

func (m *mapper) warningSubject(source string) string {
	if variable := m.optionalString(source, hydraVariable); variable != "" {
		return variable
	}
	return source
}

func (m *mapper) strings(subject, predicate string) []string {
	values := m.graph.valuesFor(subject, predicate)
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.RawValue())
	}
	return result
}

func (m *mapper) bool(subject, predicate string) bool {
	return strings.EqualFold(m.optionalString(subject, predicate), "true")
}
func (m *mapper) date(subject, predicate string) time.Time { return m.time(subject, predicate) }
func (m *mapper) time(subject, predicate string) time.Time {
	value := m.optionalString(subject, predicate)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02", "2006"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func (m *mapper) int64(subject, predicate string) int64 {
	var value int64
	if _, err := fmt.Sscan(m.optionalString(subject, predicate), &value); err != nil {
		return 0
	}
	return value
}

func (m *mapper) float64(subject, predicate string) float64 {
	var value float64
	if _, err := fmt.Sscan(m.optionalString(subject, predicate), &value); err != nil {
		return 0
	}
	return value
}

func (m *mapper) intPtr(subject, predicate string) *int64 {
	if m.optionalString(subject, predicate) == "" {
		return nil
	}
	value := m.int64(subject, predicate)
	return &value
}

func (m *mapper) floatPtr(subject, predicate string) *float64 {
	if m.optionalString(subject, predicate) == "" {
		return nil
	}
	value := m.float64(subject, predicate)
	return &value
}

func (m *mapper) validIntValues(source string, defaultValue *int64) []int64 {
	values := intValues(m.strings(source, httpParamValue))
	if defaultValue != nil && len(values) > 0 && !containsInt64(values, *defaultValue) {
		m.warnings = append(m.warnings, Diagnostic{
			Severity:  SeverityWarning,
			Subject:   m.warningSubject(source),
			Predicate: "",
			Message:   "default value is not in paramValue list; allowed values ignored",
		})
		return nil
	}
	return values
}

func (m *mapper) validFloatValues(source string, defaultValue *float64) []float64 {
	values := floatValues(m.strings(source, httpParamValue))
	if defaultValue != nil && len(values) > 0 && !containsFloat64(values, *defaultValue) {
		m.warnings = append(m.warnings, Diagnostic{
			Severity:  SeverityWarning,
			Subject:   m.warningSubject(source),
			Predicate: "",
			Message:   "default value is not in paramValue list; allowed values ignored",
		})
		return nil
	}
	return values
}

func (m *mapper) temporal(source string) *domain.Temporal {
	start := m.time(source, schemaStartDate)
	end := m.time(source, schemaEndDate)
	if start.IsZero() && end.IsZero() {
		return nil
	}
	return &domain.Temporal{Start: timePtr(start), End: timePtr(end)}
}

func (m *mapper) spatial(source string) *domain.Spatial {
	wkt := m.optionalString(source, locnGeometry)
	upper := strings.ToUpper(wkt)
	if strings.HasPrefix(upper, "POINT(") {
		values := strings.TrimSuffix(strings.TrimPrefix(wkt, "POINT("), ")")
		var longitude, latitude float64
		if _, err := fmt.Sscan(values, &longitude, &latitude); err == nil {
			return &domain.Spatial{
				Point:   &domain.Coordinate{Longitude: longitude, Latitude: latitude},
				Polygon: nil,
			}
		}
	}
	if strings.HasPrefix(upper, "POLYGON((") {
		values := strings.TrimSuffix(strings.TrimPrefix(wkt, "POLYGON(("), "))")
		coordinates := make([]domain.Coordinate, 0)
		for pair := range strings.SplitSeq(values, ",") {
			var longitude, latitude float64
			if _, err := fmt.Sscan(strings.TrimSpace(pair), &longitude, &latitude); err != nil {
				return nil
			}
			coordinates = append(coordinates, domain.Coordinate{Longitude: longitude, Latitude: latitude})
		}
		return &domain.Spatial{Point: nil, Polygon: coordinates}
	}
	return nil
}

func sharedID(kind, source string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(kind+"\x00"+source))
}

func ownedID(parent, kind, source string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(parent+"\x00"+kind+"\x00"+source))
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func intValues(values []string) []int64 {
	result := make([]int64, 0, len(values))
	for _, value := range values {
		var parsed int64
		if _, err := fmt.Sscan(value, &parsed); err == nil {
			result = append(result, parsed)
		}
	}
	return result
}

func floatValues(values []string) []float64 {
	result := make([]float64, 0, len(values))
	for _, value := range values {
		var parsed float64
		if _, err := fmt.Sscan(value, &parsed); err == nil {
			result = append(result, parsed)
		}
	}
	return result
}

func containsString(values []string, target string) bool {
	return slices.Contains(values, target)
}

func containsInt64(values []int64, target int64) bool {
	return slices.Contains(values, target)
}

func containsFloat64(values []float64, target float64) bool {
	return slices.Contains(values, target)
}

func mapValues[T any](values map[string]T) []T {
	result := make([]T, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func mapEdges(values map[string]CategoryEdge) []CategoryEdge {
	result := make([]CategoryEdge, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func mapOrganizationEdges(values map[string]OrganizationEdge) []OrganizationEdge {
	result := make([]OrganizationEdge, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
