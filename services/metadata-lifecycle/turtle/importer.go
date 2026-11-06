// Package turtle parses Turtle metadata into domain import batches.
package turtle

import (
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	rdf "github.com/deiu/rdf2go"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// TODO: try to infer decent defaults for required fields missing in the ttls but still emit the warnings so that we have valid resources as much as possible

// Parser adapts Turtle parsing to the import handler parser interface.
type Parser struct {
	log *slog.Logger
}

// New creates a TTL Parsers
func New(log *slog.Logger) *Parser {
	return &Parser{
		log: log.With("component", "turtle"),
	}
}

// Parse converts Turtle input into a metadata import batch.
func (Parser) Parse(input io.Reader) (domain.ImportBatch, error) {
	g, err := parseGraph(input, "https://epos-eu.org/metadata-lifecycle/import")
	if err != nil {
		return domain.ImportBatch{}, err
	}

	return mapGraph(g)
}

type mapper struct {
	graph             *graph
	warnings          []domain.ImportDiagnostic
	diagnosticKeys    map[string]struct{}
	invalid           map[string]struct{}
	schemes           map[string]domain.Group
	categories        map[string]domain.CategoryMetadata
	organizations     map[string]domain.OrganizationMetadata
	contacts          map[string]domain.ContactPoint
	contactKeys       map[string]string
	categoryEdges     map[string]domain.ImportCategoryEdge
	organizationEdges map[string]domain.ImportOrganizationEdge
}

func mapGraph(g *graph) (domain.ImportBatch, error) {
	m := &mapper{
		graph:             g,
		warnings:          nil,
		diagnosticKeys:    make(map[string]struct{}),
		invalid:           make(map[string]struct{}),
		schemes:           make(map[string]domain.Group),
		categories:        make(map[string]domain.CategoryMetadata),
		organizations:     make(map[string]domain.OrganizationMetadata),
		contacts:          make(map[string]domain.ContactPoint),
		contactKeys:       make(map[string]string),
		categoryEdges:     make(map[string]domain.ImportCategoryEdge),
		organizationEdges: make(map[string]domain.ImportOrganizationEdge),
	}

	var datasets []domain.DatasetMetadata
	var softwareApplications []domain.SoftwareApplicationMetadata
	var softwareSourceCodes []domain.SoftwareSourceCodeMetadata

	for _, duplicate := range g.duplicates {
		m.addWarning(domain.ImportDiagnostic{
			Severity:  domain.ImportSeverityWarning,
			Subject:   duplicate.subject,
			Predicate: duplicate.predicate,
			Message:   fmt.Sprintf("duplicate RDF statement ignored for object %q", duplicate.object),
		})
	}

	for _, source := range m.graph.subjects(rdfType, dcatDataset) {
		dataset, err := m.dataset(source)
		if err != nil {
			m.recordInvalid("dataset", source, err)
			continue
		}
		if err := dataset.Validate(); err != nil {
			m.recordInvalid("dataset", source, fmt.Errorf("validate dataset: %w", err))
			continue
		}
		if _, err := domain.NewImportBatch(domain.ImportBatch{Datasets: []domain.DatasetMetadata{dataset}}); err != nil {
			m.recordInvalid("dataset", source, err)
			continue
		}
		datasets = append(datasets, dataset)
	}

	for _, source := range m.graph.subjects(rdfType, schemaSoftwareApplication) {
		software, err := m.softwareApplication(source)
		if err != nil {
			m.recordInvalid("software application", source, err)
			continue
		}
		if err := software.Validate(); err != nil {
			m.recordInvalid("software application", source, fmt.Errorf("validate software application: %w", err))
			continue
		}
		if _, err := domain.NewImportBatch(domain.ImportBatch{SoftwareApplications: []domain.SoftwareApplicationMetadata{software}}); err != nil {
			m.recordInvalid("software application", source, err)
			continue
		}
		softwareApplications = append(softwareApplications, software)
	}

	for _, source := range m.graph.subjects(rdfType, schemaSoftwareSourceCode) {
		software, err := m.softwareSourceCode(source)
		if err != nil {
			m.recordInvalid("software source code", source, err)
			continue
		}
		if err := software.Validate(); err != nil {
			m.recordInvalid("software source code", source, fmt.Errorf("validate software source code: %w", err))
			continue
		}
		if _, err := domain.NewImportBatch(domain.ImportBatch{SoftwareSourceCodes: []domain.SoftwareSourceCodeMetadata{software}}); err != nil {
			m.recordInvalid("software source code", source, err)
			continue
		}
		softwareSourceCodes = append(softwareSourceCodes, software)
	}

	for _, source := range m.graph.subjects(rdfType, skosConceptScheme) {
		if !m.supportedScheme(source) {
			continue
		}
		if _, err := m.scheme(source); err != nil {
			m.recordInvalid("category scheme", source, err)
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
			m.recordInvalid("category", source, err)
		}
	}
	for _, source := range m.graph.subjects(rdfType, schemaOrganization) {
		if _, err := m.organization(source, make(map[string]bool)); err != nil {
			m.recordInvalid("organization", source, err)
		}
	}
	for _, source := range m.graph.subjects(rdfType, schemaContactPoint) {
		if _, err := m.contact(source); err != nil {
			m.recordInvalid("contact point", source, err)
		}
	}

	m.addUnsupportedWarnings()

	batch, err := domain.NewImportBatch(domain.ImportBatch{
		CategorySchemes:      mapValues(m.schemes),
		Categories:           mapValues(m.categories),
		Organizations:        mapValues(m.organizations),
		ContactPoints:        mapContactValues(m.contacts),
		Datasets:             datasets,
		SoftwareApplications: softwareApplications,
		SoftwareSourceCodes:  softwareSourceCodes,
		CategoryEdges:        mapEdges(m.categoryEdges),
		OrganizationEdges:    mapOrganizationEdges(m.organizationEdges),
		Warnings:             m.warnings,
	})
	if err != nil {
		return domain.ImportBatch{}, fmt.Errorf("validate import batch: %w", err)
	}
	return batch, nil
}

func (m *mapper) addWarning(warning domain.ImportDiagnostic) {
	key := string(warning.Severity) + "\x00" + warning.Subject + "\x00" + warning.Predicate + "\x00" + warning.Message
	if _, ok := m.diagnosticKeys[key]; ok {
		return
	}
	m.diagnosticKeys[key] = struct{}{}
	m.warnings = append(m.warnings, warning)
}

func (m *mapper) checkSingleValues(source string, predicates ...string) error {
	for _, predicate := range predicates {
		values := m.graph.refs(source, predicate)
		if len(values) > 1 {
			return fmt.Errorf("property %q has conflicting values", predicate)
		}
	}
	return nil
}

func (m *mapper) recordInvalid(kind, source string, err error) {
	key := kind + "\x00" + source
	if _, ok := m.invalid[key]; ok {
		return
	}
	m.invalid[key] = struct{}{}
	m.addWarning(domain.ImportDiagnostic{
		Severity: domain.ImportSeverityError,
		Subject:  source,
		Message:  fmt.Sprintf("%s skipped: %s", kind, err),
	})
}

func (m *mapper) isInvalid(kind, source string) bool {
	_, ok := m.invalid[kind+"\x00"+source]
	return ok
}

func (m *mapper) omitRelation(owner, predicate, kind, source string, err error) {
	m.addWarning(domain.ImportDiagnostic{
		Severity:  domain.ImportSeverityWarning,
		Subject:   owner,
		Predicate: predicate,
		Message:   fmt.Sprintf("%s %q omitted: %s", kind, source, err),
	})
}

func (m *mapper) contactValue(source, owner, predicate string) (domain.ContactPoint, bool) {
	if m.isInvalid("contact point", source) {
		m.omitRelation(owner, predicate, "contact point", source, fmt.Errorf("contact point is invalid"))
		return domain.ContactPoint{}, false
	}
	value, err := m.contact(source)
	if err != nil {
		m.recordInvalid("contact point", source, err)
		m.omitRelation(owner, predicate, "contact point", source, err)
		return domain.ContactPoint{}, false
	}
	return value, true
}

func (m *mapper) organizationValue(source, owner, predicate string) (domain.OrganizationMetadata, bool) {
	if m.isInvalid("organization", source) {
		m.omitRelation(owner, predicate, "organization", source, fmt.Errorf("organization is invalid"))
		return domain.OrganizationMetadata{}, false
	}
	value, err := m.organization(source, make(map[string]bool))
	if err != nil {
		m.recordInvalid("organization", source, err)
		m.omitRelation(owner, predicate, "organization", source, err)
		return domain.OrganizationMetadata{}, false
	}
	return value, true
}

func (m *mapper) categoryValue(source, owner, predicate string) (domain.CategoryMetadata, bool) {
	if m.isInvalid("category", source) {
		m.omitRelation(owner, predicate, "category", source, fmt.Errorf("category is invalid"))
		return domain.CategoryMetadata{}, false
	}
	value, err := m.category(source, make(map[string]bool))
	if err != nil {
		m.recordInvalid("category", source, err)
		m.omitRelation(owner, predicate, "category", source, err)
		return domain.CategoryMetadata{}, false
	}
	return value, true
}

func (m *mapper) softwareApplication(source string) (domain.SoftwareApplicationMetadata, error) {
	if err := m.checkSingleValues(source,
		schemaName, schemaDescription, schemaMainEntityOfPage, schemaLicense,
		schemaSoftwareVersion, schemaVersion, schemaDownloadURL,
		schemaSoftwareRequirements, schemaOperatingSystem,
	); err != nil {
		return domain.SoftwareApplicationMetadata{}, err
	}
	name := m.requiredString(source, schemaName)
	result := domain.SoftwareApplicationMetadata{
		ID:               sharedID("software-application", source),
		Name:             name,
		Description:      m.importedDescription(source, schemaDescription, "software application", name),
		MainEntityOfPage: stringPtr(m.optionalString(source, schemaMainEntityOfPage)),
		License:          stringPtr(m.optionalString(source, schemaLicense)),
		Version:          stringPtr(m.optionalStringOneOf(source, schemaSoftwareVersion, schemaVersion)),
		Keywords:         m.keywords(source, schemaKeywords),
		DownloadURL:      stringPtr(m.optionalString(source, schemaDownloadURL)),
		Requirements:     stringPtr(m.optionalString(source, schemaSoftwareRequirements)),
		OperatingSystem:  stringPtr(m.optionalString(source, schemaOperatingSystem)),
		Creators:         nil,
		Identifiers:      nil,
		ContactPoints:    nil,
		Categories:       nil,
	}

	if err := m.addSoftwareRelations(source, result.ID, &result.Creators, &result.Identifiers, &result.ContactPoints, &result.Categories); err != nil {
		return domain.SoftwareApplicationMetadata{}, err
	}

	return result, nil
}

func (m *mapper) softwareSourceCode(source string) (domain.SoftwareSourceCodeMetadata, error) {
	if err := m.checkSingleValues(source,
		schemaName, schemaDescription, schemaProgrammingLanguage, schemaLicense,
		schemaCodeRepository, schemaSoftwareRequirements, schemaRuntimePlatform,
		schemaMainEntityOfPage, schemaVersion,
	); err != nil {
		return domain.SoftwareSourceCodeMetadata{}, err
	}
	name := m.requiredString(source, schemaName)
	result := domain.SoftwareSourceCodeMetadata{
		ID:                   sharedID("software-source-code", source),
		Name:                 name,
		Description:          m.importedDescription(source, schemaDescription, "software source code", name),
		ProgrammingLanguage:  stringPtr(m.optionalString(source, schemaProgrammingLanguage)),
		Keywords:             m.keywords(source, schemaKeywords),
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
		return domain.SoftwareSourceCodeMetadata{}, err
	}

	return result, nil
}

func (m *mapper) addSoftwareRelations(
	source string,
	owner uuid.UUID,
	creators *[]domain.OrganizationMetadata,
	identifiers *[]domain.Identifier,
	contacts *[]domain.ContactPoint,
	categories *[]domain.CategoryMetadata,
) error {
	for _, identifierSource := range m.graph.refs(source, schemaIdentifier) {
		if identifier, ok := m.identifier(identifierSource, owner, source); ok {
			*identifiers = appendUniqueIdentifier(*identifiers, identifier)
		}
	}

	for _, creatorSource := range m.graph.refs(source, schemaCreator) {
		creator, ok := m.organizationValue(creatorSource, source, schemaCreator)
		if !ok {
			continue
		}
		*creators = append(*creators, creator)
	}

	for _, contactSource := range m.graph.refs(source, dcatContactPoint) {
		contact, ok := m.contactValue(contactSource, source, dcatContactPoint)
		if !ok {
			continue
		}
		*contacts = appendUniqueContactPoint(*contacts, contact)
	}

	for _, categorySource := range m.graph.refs(source, dcatTheme) {
		category, ok := m.categoryValue(categorySource, source, dcatTheme)
		if !ok {
			continue
		}
		*categories = append(*categories, category)
	}

	return nil
}

func (m *mapper) dataset(source string) (domain.DatasetMetadata, error) {
	if err := m.checkSingleValues(source,
		dctTitle, dctDescription, dctAccrualPeriodicity, dqvHasQualityAnnotation,
		dctAccessRights, dctCreated, dctModified, dctTemporal, dctSpatial,
	); err != nil {
		return domain.DatasetMetadata{}, err
	}
	title := m.requiredString(source, dctTitle)
	description := m.importedDescription(source, dctDescription, "dataset", title)
	id := ownedID("", "dataset", source)
	dataset := domain.DatasetMetadata{
		ID:          id,
		Title:       title,
		Description: description,
		Keywords:    m.keywords(source, dcatKeyword),

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
		SourceCreated:         nil,
		SourceUpdated:         nil,
	}

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
			dataset.PersistentIdentifiers = appendUniqueIdentifier(dataset.PersistentIdentifiers, identifier)
		}
	}

	if value, ok := m.first(source, dctTemporal); ok {
		dataset.TemporalCoverage = m.temporal(value.RawValue())
	}

	if value, ok := m.first(source, dctSpatial); ok {
		dataset.SpatialCoverage = m.spatial(value.RawValue())
	}

	for _, ref := range m.graph.refs(source, dcatTheme) {
		category, ok := m.categoryValue(ref, source, dcatTheme)
		if !ok {
			continue
		}
		dataset.Categories = append(dataset.Categories, category)
	}

	for _, ref := range m.graph.refs(source, dcatContactPoint) {
		contact, ok := m.contactValue(ref, source, dcatContactPoint)
		if !ok {
			continue
		}
		dataset.ContactPoints = appendUniqueContactPoint(dataset.ContactPoints, contact)
	}

	for _, ref := range m.graph.refs(source, dctPublisher) {
		organization, ok := m.organizationValue(ref, source, dctPublisher)
		if !ok {
			continue
		}
		dataset.Providers = append(dataset.Providers, organization)
	}

	for _, ref := range m.graph.refs(source, dcatDistributionLink) {
		distribution, err := m.distribution(ref, id)
		if err != nil {
			m.recordInvalid("distribution", ref, err)
			m.omitRelation(source, dcatDistributionLink, "distribution", ref, err)
			continue
		}
		dataset.Distributions = append(dataset.Distributions, distribution)
	}

	return dataset, nil
}

func (m *mapper) importedDescription(source, predicate, kind, title string) string {
	description, ok := m.firstString(source, predicate)
	if ok && strings.TrimSpace(description) != "" {
		return description
	}

	description = fmt.Sprintf("%s description", strings.TrimSpace(title))
	m.addWarning(domain.ImportDiagnostic{
		Severity:  domain.ImportSeverityWarning,
		Subject:   source,
		Predicate: predicate,
		Message:   fmt.Sprintf("%s description missing; using default %q", kind, description),
	})
	return description
}

func (m *mapper) distribution(source string, owner uuid.UUID) (domain.Distribution, error) {
	if err := m.checkSingleValues(source,
		dctTitle, dctDescription, dctLicense, dctIssued, dctModified,
		dctType, dcatByteSize, dctConformsTo, dctFormat, dcatDownloadURL, dcatAccessURL,
	); err != nil {
		return domain.Distribution{}, err
	}
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

	distributionType := normalizeDistributionType(m.optionalString(source, dctType))
	serviceSource, hasService := m.firstString(source, dctConformsTo)
	switch distributionType {
	case domain.DistributionTypeWebService:
		if !hasService {
			return domain.Distribution{}, fmt.Errorf("web service distribution requires %s", dctConformsTo)
		}
		webService, err := m.webServiceForDistribution(source, serviceSource, distribution.ID)
		if err != nil {
			m.recordInvalid("web service", serviceSource, err)
			return domain.Distribution{}, fmt.Errorf("web service %q: %w", serviceSource, err)
		}
		distribution.WebService = &webService
	case domain.DistributionTypeDownloadableFile:
		distribution.DownloadableFile = downloadableFile(m, source)
	case "":
		if hasService && m.graph.hasType(serviceSource, "https://www.epos-eu.org/epos-dcat-ap#WebService") {
			webService, err := m.webServiceForDistribution(source, serviceSource, distribution.ID)
			if err != nil {
				m.recordInvalid("web service", serviceSource, err)
				return domain.Distribution{}, fmt.Errorf("web service %q: %w", serviceSource, err)
			}
			distribution.WebService = &webService
		} else {
			distribution.DownloadableFile = downloadableFile(m, source)
		}
	default:
		return domain.Distribution{}, fmt.Errorf("unsupported distribution type %q: %w", distributionType, apperror.Default(apperror.CodeValidationFailed))
	}

	typeValue, err := distribution.DeliveryType()
	if err != nil {
		return domain.Distribution{}, fmt.Errorf("distribution %q: %w", source, err)
	}

	distribution.Type = typeValue
	if distribution.WebService != nil {
		if _, err := domain.NewWebServiceInput(distribution.WebService.Input()); err != nil {
			m.recordInvalid("web service", serviceSource, err)
			return domain.Distribution{}, fmt.Errorf("web service %q is invalid: %w", serviceSource, err)
		}
	}
	input := domain.DistributionInput{
		Title:            distribution.Title,
		Description:      distribution.Description,
		License:          distribution.License,
		Type:             distribution.Type,
		Issued:           distribution.Issued,
		Modified:         distribution.Modified,
		ByteSize:         distribution.ByteSize,
		DownloadableFile: distribution.DownloadableFile,
	}
	if distribution.WebService != nil {
		webServiceInput := distribution.WebService.Input()
		input.WebService = &webServiceInput
	}
	if _, err := domain.NewDistributionInput(input); err != nil {
		return domain.Distribution{}, fmt.Errorf("distribution %q is invalid: %w", source, err)
	}

	return distribution, nil
}

func downloadableFile(m *mapper, source string) *domain.DownloadableFile {
	url := m.optionalString(source, dcatDownloadURL)
	if url == "" {
		url = m.optionalString(source, dcatAccessURL)
	}
	if url == "" {
		return nil
	}
	return &domain.DownloadableFile{URL: url, Format: m.optionalString(source, dctFormat)}
}

func (m *mapper) webServiceForDistribution(distribution, source string, owner uuid.UUID) (domain.WebService, error) {
	operation, ok := m.firstString(distribution, dcatAccessURL)
	if !ok || strings.TrimSpace(operation) == "" {
		return domain.WebService{}, fmt.Errorf("web service distribution requires %s", dcatAccessURL)
	}
	return m.webService(source, owner, operation)
}

func (m *mapper) webService(source string, owner uuid.UUID, operationSource string) (domain.WebService, error) {
	if err := m.checkSingleValues(source,
		schemaName, schemaDescription, dctTitle, dctTemporal, dctSpatial,
	); err != nil {
		return domain.WebService{}, err
	}
	if err := m.checkSingleValues(operationSource, hydraReturns); err != nil {
		return domain.WebService{}, err
	}
	configuration := domain.GenericWebServiceConfiguration{
		Method: domain.HTTPMethodGet,
		Format: m.optionalString(operationSource, hydraReturns),
	}
	service := domain.WebService{
		ID:          ownedID(owner.String(), "web-service", source),
		Name:        m.optionalString(source, schemaName),
		Description: stringPtr(m.optionalString(source, schemaDescription)),
		Type:        domain.WebServiceTypeHTTP,
		// TODO: Import visible-on targets from source metadata.
		VisibleOn:        nil,
		Configuration:    configuration,
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
	method := strings.ToUpper(m.optionalString(operationSource, hydraMethod))
	if method == "" {
		method = string(domain.HTTPMethodGet)
	}
	configuration.Method = domain.HTTPMethod(method)
	if !configuration.Method.Valid() {
		return domain.WebService{}, fmt.Errorf("unsupported web service method %q", method)
	}

	providers := m.graph.refs(source, schemaProvider)
	if len(providers) > 1 {
		return domain.WebService{}, fmt.Errorf("web service must have at most one provider")
	}
	if len(providers) == 1 {
		item, ok := m.organizationValue(providers[0], source, schemaProvider)
		if ok {
			service.Provider = &item
		}
	}

	for _, contact := range m.graph.refs(source, dcatContactPoint) {
		item, ok := m.contactValue(contact, source, dcatContactPoint)
		if !ok {
			continue
		}
		service.ContactPoints = appendUniqueContactPoint(service.ContactPoints, item)
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

	serviceTemplate := ""
	if rawTemplate, ok := m.firstString(operationSource, hydraTemplate); ok {
		serviceTemplate = rawTemplate
		configuration.TemplateURL = stripHydraQueryTemplate(rawTemplate)
	}
	for _, property := range m.graph.refs(operationSource, hydraProperty) {
		if rawTemplate, ok := m.firstString(property, hydraTemplate); ok && serviceTemplate == "" {
			serviceTemplate = rawTemplate
			configuration.TemplateURL = stripHydraQueryTemplate(rawTemplate)
		}
		for _, mapping := range m.graph.refs(property, hydraMapping) {
			parameterValue, err := m.parameter(mapping)
			if err != nil {
				m.recordInvalid("parameter", mapping, err)
				m.omitRelation(source, hydraMapping, "parameter", mapping, err)
				continue
			}
			parameterValue.Position = len(configuration.Parameters)
			propertyValue := m.optionalString(mapping, hydraProperty)
			parameterValue.Property = stringPtr(propertyValue)
			parameterValue.Role = webServiceParameterRole(propertyValue)
			parameter := domain.WebServiceParameter{
				ParameterInput: parameterValue,
				ID:             ownedID(service.ID.String(), "parameter", mapping),
				Location:       hydraParameterLocation(serviceTemplate, parameterValue.Name),
				Explode:        false,
			}
			parameters, mergeErr := appendUniqueParameter(configuration.Parameters, parameter)
			if mergeErr != nil {
				m.recordInvalid("parameter", mapping, mergeErr)
				m.omitRelation(source, hydraMapping, "parameter", mapping, mergeErr)
				continue
			}
			configuration.Parameters = parameters
		}
	}
	service.Configuration = configuration

	return service, nil
}

func webServiceParameterRole(property string) *domain.WebServiceParameterRole {
	property = strings.TrimSpace(property)
	var role domain.WebServiceParameterRole
	switch property {
	case "schema:encodingFormat":
		role = domain.WebServiceParameterRoleFormat
	case "schema:startDate":
		role = domain.WebServiceParameterRoleStartDate
	case "schema:endDate":
		role = domain.WebServiceParameterRoleEndDate
	case "epos:westernmostLongitude":
		role = domain.WebServiceParameterRoleMinLon
	case "epos:easternmostLongitude":
		role = domain.WebServiceParameterRoleMaxLon
	case "epos:southernmostLatitude":
		role = domain.WebServiceParameterRoleMinLat
	case "epos:northernmostLatitude":
		role = domain.WebServiceParameterRoleMaxLat
	default:
		return nil
	}
	return &role
}

func (m *mapper) parameter(source string) (domain.ParameterInput, error) {
	if err := m.checkSingleValues(source,
		hydraVariable, "http://www.w3.org/2000/01/rdf-schema#label", hydraProperty,
		hydraRequired, schemaMultipleValues, schemaReadonlyValue,
		rdfsRange, schemaDefaultValue, schemaMinValue, schemaMaxValue,
		schemaValuePattern,
	); err != nil {
		return domain.ParameterInput{}, err
	}
	name := m.requiredString(source, hydraVariable)
	label := m.optionalString(source, "http://www.w3.org/2000/01/rdf-schema#label")
	if label == "" {
		label = name
	}
	if name == "" {
		return domain.ParameterInput{}, fmt.Errorf("parameter %q has no variable", source)
	}
	parameter := domain.ParameterInput{
		Name:     name,
		Label:    label,
		Required: m.bool(source, hydraRequired),
		ReadOnly: m.bool(source, schemaReadonlyValue),
	}
	multipleValues := m.bool(source, schemaMultipleValues)
	rangeIRI := normalizeRange(m.optionalString(source, rdfsRange))
	config := func() (string, []string) {
		return m.optionalString(source, schemaDefaultValue), m.strings(source, httpParamValue)
	}
	switch rangeIRI {
	case rdfsDatatypeBoolean:
		defaultText, hasDefault := m.firstString(source, schemaDefaultValue)
		defaultValue := strings.EqualFold(defaultText, "true")
		if hasDefault && !strings.EqualFold(defaultText, "false") && !defaultValue {
			return domain.ParameterInput{}, fmt.Errorf("invalid boolean default %q", defaultText)
		}
		var defaultPointer *bool
		if hasDefault {
			defaultPointer = &defaultValue
		}
		parameter.Schema = domain.BooleanSchema{Default: defaultPointer}
		return parameter, nil
	case rdfsDatatypeInteger:
		defaultValue := m.intPtr(source, schemaDefaultValue)
		var defaults []int64
		if defaultValue != nil {
			defaults = []int64{*defaultValue}
		}
		parameter.Schema = domain.IntegerSchema{
			Minimum:        m.intPtr(source, schemaMinValue),
			Maximum:        m.intPtr(source, schemaMaxValue),
			Default:        defaults,
			Enum:           intEnumOptions(m.validIntValues(source, defaultValue)),
			MultipleValues: multipleValues,
		}
		return parameter, nil
	case rdfsDatatypeFloat:
		defaultValue := m.floatPtr(source, schemaDefaultValue)
		var defaults []float64
		if defaultValue != nil {
			defaults = []float64{*defaultValue}
		}
		parameter.Schema = domain.FloatSchema{
			Minimum:        m.floatPtr(source, schemaMinValue),
			Maximum:        m.floatPtr(source, schemaMaxValue),
			Default:        defaults,
			Enum:           floatEnumOptions(m.validFloatValues(source, defaultValue)),
			MultipleValues: multipleValues,
		}
		return parameter, nil
	case rdfsDatatypeDateTime:
		parameter.Schema = domain.DateTimeSchema{
			Minimum: timePtr(m.time(source, schemaMinValue)),
			Maximum: timePtr(m.time(source, schemaMaxValue)),
			Default: timePtr(m.time(source, schemaDefaultValue)),
			Layout:  stringPtr(m.optionalString(source, schemaValuePattern)),
		}
		return parameter, nil
	default:
		defaultValue, allowed := config()
		var pointer *string
		if defaultValue != "" {
			pointer = &defaultValue
		}
		if pointer != nil && len(allowed) > 0 && !containsString(allowed, *pointer) {
			m.addWarning(domain.ImportDiagnostic{
				Severity:  domain.ImportSeverityWarning,
				Subject:   m.warningSubject(source),
				Predicate: "",
				Message:   "default value is not in paramValue list; allowed values ignored",
			})
			allowed = nil
		}
		var defaults []string
		if pointer != nil {
			defaults = []string{*pointer}
		}
		parameter.Schema = domain.StringSchema{
			Default:        defaults,
			Enum:           stringEnumOptions(allowed),
			Pattern:        stringPtr(m.optionalString(source, schemaValuePattern)),
			MultipleValues: multipleValues,
		}
		return parameter, nil
	}
}

func (m *mapper) scheme(source string) (domain.Group, error) {
	if err := m.checkSingleValues(source,
		dctTitle, dctDescription, skosPrefLabel, foafLogo, foafHomepage,
		schemaColor, schemaOrderItemNumber, skosHasTopConcept,
	); err != nil {
		return domain.Group{}, err
	}
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
	result := domain.Group{
		ID:          sharedID("category-scheme", source),
		Name:        m.optionalString(source, dctTitle),
		Description: stringPtr(m.optionalString(source, dctDescription)),
		Label:       stringPtr(m.optionalString(source, skosPrefLabel)),
		Logo:        stringPtr(m.optionalString(source, foafLogo)),
		Page:        stringPtr(m.optionalString(source, foafHomepage)),
		Color:       stringPtr(m.optionalString(source, schemaColor)),
		Position:    int32(m.int64(source, schemaOrderItemNumber)),
		Concept:     "",
	}
	if result.Name == "" {
		result.Name = m.optionalString(source, skosPrefLabel)
	}
	result.Concept = string(concept)
	if !concept.Valid() {
		err := fmt.Errorf("category scheme %q has unsupported top concept", source)
		m.recordInvalid("category scheme", source, err)
		return domain.Group{}, err
	}
	if _, err := domain.NewGroup(result.ID, domain.GroupInput{
		Name:        result.Name,
		Description: result.Description,
		Label:       result.Label,
		Logo:        result.Logo,
		Page:        result.Page,
		Color:       result.Color,
		Position:    result.Position,
		Concept:     result.Concept,
	}); err != nil {
		err = fmt.Errorf("category scheme %q is invalid: %w", source, err)
		m.recordInvalid("category scheme", source, err)
		return domain.Group{}, err
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

func (m *mapper) category(source string, active map[string]bool) (domain.CategoryMetadata, error) {
	if active[source] {
		return domain.CategoryMetadata{}, fmt.Errorf("category hierarchy cycle at %q", source)
	}
	if value, ok := m.categories[source]; ok {
		return value, nil
	}
	active[source] = true
	defer delete(active, source)
	if err := m.checkSingleValues(source, skosInScheme, skosPrefLabel); err != nil {
		return domain.CategoryMetadata{}, err
	}
	schemeIRI, ok := m.firstString(source, skosInScheme)
	if !ok {
		return domain.CategoryMetadata{}, fmt.Errorf("category %q has no scheme", source)
	}
	scheme, err := m.scheme(schemeIRI)
	if err != nil {
		return domain.CategoryMetadata{}, err
	}
	result := domain.CategoryMetadata{
		ID:       sharedID("category", source),
		Name:     m.requiredString(source, skosPrefLabel),
		Scheme:   scheme,
		Children: nil,
	}
	parents := m.graph.refs(source, skosBroader)
	if len(parents) > 1 {
		return domain.CategoryMetadata{}, fmt.Errorf("category %q has multiple parents", source)
	}
	for _, parentIRI := range parents {
		if active[parentIRI] {
			m.omitRelation(source, skosBroader, "category parent", parentIRI, fmt.Errorf("category hierarchy cycle"))
			continue
		}
		parent, err := m.category(parentIRI, active)
		if err != nil {
			m.recordInvalid("category", parentIRI, err)
			m.omitRelation(source, skosBroader, "category parent", parentIRI, err)
			continue
		}
		m.categoryEdges[edgeKey(parent.ID, result.ID)] = domain.ImportCategoryEdge{ParentID: parent.ID, ChildID: result.ID}
	}
	if err := result.Validate(); err != nil {
		return domain.CategoryMetadata{}, fmt.Errorf("category %q is invalid: %w", source, err)
	}
	m.categories[source] = result
	return result, nil
}

func (m *mapper) organization(source string, active map[string]bool) (domain.OrganizationMetadata, error) {
	if active[source] {
		return domain.OrganizationMetadata{}, fmt.Errorf("organization hierarchy cycle at %q", source)
	}
	if value, ok := m.organizations[source]; ok {
		return value, nil
	}
	active[source] = true
	defer delete(active, source)
	if err := m.checkSingleValues(source,
		schemaLegalName, schemaName, schemaAddress, schemaLogo, schemaURL,
		schemaEmail, schemaLEICode,
	); err != nil {
		return domain.OrganizationMetadata{}, err
	}
	address := source
	if references := m.graph.refs(source, schemaAddress); len(references) > 0 {
		address = references[0]
	}
	result := domain.OrganizationMetadata{
		ID:            sharedID("organization", source),
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
		return domain.OrganizationMetadata{}, fmt.Errorf("organization %q has no name", source)
	}
	for _, identifierSource := range m.graph.refs(source, schemaIdentifier) {
		if identifier, ok := m.identifier(identifierSource, result.ID, source); ok {
			result.Identifiers = appendUniqueIdentifier(result.Identifiers, identifier)
		}
	}
	for _, contactSource := range m.graph.refs(source, dcatContactPoint) {
		contact, ok := m.contactValue(contactSource, source, dcatContactPoint)
		if !ok {
			continue
		}
		result.ContactPoints = appendUniqueContactPoint(result.ContactPoints, contact)
	}
	for _, parentIRI := range m.graph.refs(source, schemaMemberOf) {
		if active[parentIRI] {
			m.omitRelation(source, schemaMemberOf, "organization parent", parentIRI, fmt.Errorf("organization hierarchy cycle"))
			continue
		}
		parent, err := m.organization(parentIRI, active)
		if err != nil {
			m.recordInvalid("organization", parentIRI, err)
			m.omitRelation(source, schemaMemberOf, "organization parent", parentIRI, err)
			continue
		}
		m.organizationEdges[edgeKey(parent.ID, result.ID)] = domain.ImportOrganizationEdge{ParentID: parent.ID, ChildID: result.ID}
	}
	if err := result.Validate(); err != nil {
		return domain.OrganizationMetadata{}, fmt.Errorf("organization %q is invalid: %w", source, err)
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

func normalizeDistributionType(value string) domain.DistributionType {
	switch strings.TrimSpace(value) {
	case string(domain.DistributionTypeWebService),
		"http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE",
		"https://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE":
		return domain.DistributionTypeWebService
	case string(domain.DistributionTypeDownloadableFile),
		"http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE",
		"https://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE":
		return domain.DistributionTypeDownloadableFile
	default:
		return domain.DistributionType(value)
	}
}

func (m *mapper) contact(source string) (domain.ContactPoint, error) {
	if value, ok := m.contacts[source]; ok {
		return value, nil
	}
	if err := m.checkSingleValues(source, schemaEmail, schemaContactType); err != nil {
		return domain.ContactPoint{}, err
	}
	email := m.optionalString(source, schemaEmail)
	role := m.optionalString(source, schemaContactType)
	if email == "" || role == "" {
		return domain.ContactPoint{}, fmt.Errorf("contact point %q requires email and role", source)
	}
	result, err := domain.NewContactPoint(sharedID("contact-point", source), domain.ContactPointInput{
		Email: email,
		Role:  role,
	})
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("contact point %q is invalid: %w", source, err)
	}
	key := result.Email + "\x00" + result.Role
	if canonicalSource, ok := m.contactKeys[key]; ok {
		result = m.contacts[canonicalSource]
		m.addWarning(domain.ImportDiagnostic{
			Severity: domain.ImportSeverityWarning,
			Subject:  source,
			Message:  fmt.Sprintf("contact point %q merged with %q", source, canonicalSource),
		})
	} else {
		m.contactKeys[key] = source
	}
	m.contacts[source] = result
	return result, nil
}

func (m *mapper) identifiers(subject string, owner uuid.UUID) []domain.Identifier {
	result := make([]domain.Identifier, 0)
	for _, identifierSource := range m.graph.refs(subject, dctIdentifier) {
		if identifier, ok := m.identifier(identifierSource, owner, subject); ok {
			result = appendUniqueIdentifier(result, identifier)
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
		return domain.Identifier{ID: uuid.Nil, Type: "", Value: ""}, false
	}
	key := source
	if ownerIRI != "" {
		key = ownerIRI + "\x00" + typ + "\x00" + value
	}
	return domain.Identifier{
		ID:    ownedID(owner.String(), "identifier", key),
		Type:  domain.IdentifierType(typ),
		Value: value,
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
				m.addWarning(domain.ImportDiagnostic{
					Severity:  domain.ImportSeverityWarning,
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

func (m *mapper) keywords(subject, predicate string) []string {
	result := make([]string, 0)
	for _, value := range m.strings(subject, predicate) {
		for keyword := range strings.SplitSeq(value, ",") {
			result = append(result, strings.TrimSpace(keyword))
		}
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
	return new(value)
}

func (m *mapper) floatPtr(subject, predicate string) *float64 {
	if m.optionalString(subject, predicate) == "" {
		return nil
	}
	value := m.float64(subject, predicate)
	return new(value)
}

func (m *mapper) validIntValues(source string, defaultValue *int64) []int64 {
	values := intValues(m.strings(source, httpParamValue))
	if defaultValue != nil && len(values) > 0 && !containsInt64(values, *defaultValue) {
		m.addWarning(domain.ImportDiagnostic{
			Severity:  domain.ImportSeverityWarning,
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
		m.addWarning(domain.ImportDiagnostic{
			Severity:  domain.ImportSeverityWarning,
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

func (m *mapper) spatial(source string) geom.T {
	value := strings.TrimSpace(m.optionalString(source, locnGeometry))
	if value == "" {
		return nil
	}
	geometry, err := wkt.Unmarshal(value)
	if err != nil {
		return nil
	}
	geometry, err = geom.SetSRID(geometry, 4326)
	if err != nil {
		return nil
	}
	return geometry
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
	return new(value)
}

func stripHydraQueryTemplate(value string) string {
	for _, marker := range []string{"{?", "{&"} {
		if before, _, ok := strings.Cut(value, marker); ok {
			return before
		}
	}
	return value
}

// TODO: Parse hydra template correctly
func hydraParameterLocation(template, name string) domain.ParameterLocation {
	for _, marker := range []string{"{?", "{&"} {
		start := 0
		for {
			index := strings.Index(template[start:], marker)
			if index < 0 {
				break
			}
			index += start
			end := strings.IndexByte(template[index:], '}')
			if end < 0 {
				break
			}
			end += index
			for variable := range strings.SplitSeq(template[index+2:end], ",") {
				if strings.TrimSpace(variable) == name {
					return domain.ParameterLocationQuery
				}
			}
			start = end + 1
		}
	}
	if strings.Contains(template, "{"+name+"}") {
		return domain.ParameterLocationPath
	}
	return domain.ParameterLocationQuery
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return new(value)
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

func appendUniqueContactPoint(values []domain.ContactPoint, value domain.ContactPoint) []domain.ContactPoint {
	for _, existing := range values {
		if existing.ID == value.ID {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueIdentifier(values []domain.Identifier, value domain.Identifier) []domain.Identifier {
	for _, existing := range values {
		if existing.Type == value.Type && existing.Value == value.Value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueParameter(values []domain.WebServiceParameter, value domain.WebServiceParameter) ([]domain.WebServiceParameter, error) {
	for i, existing := range values {
		if existing.Name != value.Name {
			continue
		}
		if existing.Schema == nil || value.Schema == nil || existing.Schema.Kind() != value.Schema.Kind() {
			return values, fmt.Errorf("parameter %q has incompatible kinds %q and %q", value.Name, schemaKind(existing.Schema), schemaKind(value.Schema))
		}
		if existing.Role != nil && value.Role != nil && *existing.Role != *value.Role {
			return values, fmt.Errorf("parameter %q has conflicting roles %q and %q", value.Name, *existing.Role, *value.Role)
		}
		values[i].ParameterInput = mergeParameter(existing.ParameterInput, value.ParameterInput)
		if values[i].Role == nil {
			values[i].Role = value.Role
		}
		return values, nil
	}
	return append(values, value), nil
}

func mergeParameter(existing, incoming domain.ParameterInput) domain.ParameterInput {
	merged := existing
	merged.Description = mergeStringDefault(existing.Description, incoming.Description)
	merged.Required = existing.Required || incoming.Required
	merged.ReadOnly = existing.ReadOnly || incoming.ReadOnly

	switch current := existing.Schema.(type) {
	case domain.StringSchema:
		other, ok := incoming.Schema.(domain.StringSchema)
		if !ok {
			return merged
		}
		current.MultipleValues = current.MultipleValues || other.MultipleValues
		current.Enum = nilIfEmpty(mergeStringValues(current.Enum, other.Enum))
		current.Default = mergeSliceDefault(current.Default, other.Default)
		if !sameStringPointer(current.Pattern, other.Pattern) {
			current.Pattern = nil
		}
		merged.Schema = current
	case domain.BooleanSchema:
		other, ok := incoming.Schema.(domain.BooleanSchema)
		if ok {
			current.Default = mergeBoolDefault(current.Default, other.Default)
		}
		merged.Schema = current
	case domain.IntegerSchema:
		other, ok := incoming.Schema.(domain.IntegerSchema)
		if !ok {
			return merged
		}
		current.MultipleValues = current.MultipleValues || other.MultipleValues
		current.Minimum = mergeIntDefault(current.Minimum, other.Minimum)
		current.Maximum = mergeIntDefault(current.Maximum, other.Maximum)
		current.Default = mergeSliceDefault(current.Default, other.Default)
		current.Enum = nilIfEmpty(mergeIntValues(current.Enum, other.Enum))
		merged.Schema = current
	case domain.FloatSchema:
		other, ok := incoming.Schema.(domain.FloatSchema)
		if !ok {
			return merged
		}
		current.MultipleValues = current.MultipleValues || other.MultipleValues
		current.Minimum = mergeFloatDefault(current.Minimum, other.Minimum)
		current.Maximum = mergeFloatDefault(current.Maximum, other.Maximum)
		current.Default = mergeSliceDefault(current.Default, other.Default)
		current.Enum = nilIfEmpty(mergeFloatValues(current.Enum, other.Enum))
		merged.Schema = current
	case domain.DateTimeSchema:
		other, ok := incoming.Schema.(domain.DateTimeSchema)
		if !ok {
			return merged
		}
		current.Minimum = mergeTimeDefault(current.Minimum, other.Minimum)
		current.Maximum = mergeTimeDefault(current.Maximum, other.Maximum)
		current.Default = mergeTimeDefault(current.Default, other.Default)
		current.Layout = mergeStringDefault(current.Layout, other.Layout)
		merged.Schema = current
	}
	return merged
}

func schemaKind(schema domain.ParameterSchema) domain.ParameterSchemaKind {
	if schema == nil {
		return ""
	}
	return schema.Kind()
}

func sameStringPointer(first, second *string) bool {
	if first == nil || second == nil {
		return first == second
	}
	return *first == *second
}

func nilIfEmpty[T any](values []T) []T {
	if len(values) == 0 {
		return nil
	}
	return values
}

func stringEnumOptions(values []string) []domain.StringEnumOption {
	if len(values) == 0 {
		return nil
	}
	result := make([]domain.StringEnumOption, len(values))
	for i, value := range values {
		result[i] = domain.StringEnumOption{Value: value}
	}
	return result
}

func intEnumOptions(values []int64) []domain.IntegerEnumOption {
	if len(values) == 0 {
		return nil
	}
	result := make([]domain.IntegerEnumOption, len(values))
	for i, value := range values {
		result[i] = domain.IntegerEnumOption{Value: value}
	}
	return result
}

func floatEnumOptions(values []float64) []domain.FloatEnumOption {
	if len(values) == 0 {
		return nil
	}
	result := make([]domain.FloatEnumOption, len(values))
	for i, value := range values {
		result[i] = domain.FloatEnumOption{Value: value}
	}
	return result
}

func mergeStringValues(first, second []domain.StringEnumOption) []domain.StringEnumOption {
	result := append([]domain.StringEnumOption(nil), first...)
	for _, value := range second {
		found := false
		for i := range result {
			if result[i].Value != value.Value {
				continue
			}
			if result[i].Label == nil {
				result[i].Label = value.Label
			}
			found = true
			break
		}
		if !found {
			result = append(result, value)
		}
	}
	return result
}

func mergeIntValues(first, second []domain.IntegerEnumOption) []domain.IntegerEnumOption {
	result := append([]domain.IntegerEnumOption(nil), first...)
	for _, value := range second {
		found := false
		for i := range result {
			if result[i].Value != value.Value {
				continue
			}
			if result[i].Label == nil {
				result[i].Label = value.Label
			}
			found = true
			break
		}
		if !found {
			result = append(result, value)
		}
	}
	return result
}

func mergeFloatValues(first, second []domain.FloatEnumOption) []domain.FloatEnumOption {
	result := append([]domain.FloatEnumOption(nil), first...)
	for _, value := range second {
		found := false
		for i := range result {
			if result[i].Value != value.Value {
				continue
			}
			if result[i].Label == nil {
				result[i].Label = value.Label
			}
			found = true
			break
		}
		if !found {
			result = append(result, value)
		}
	}
	return result
}

func mergeStringDefault(first, second *string) *string {
	if first == nil {
		return second
	}
	if second == nil || *first == *second {
		return first
	}
	return nil
}

func mergeSliceDefault[T comparable](first, second []T) []T {
	if first == nil {
		return second
	}
	if second == nil || slices.Equal(first, second) {
		return first
	}
	return nil
}

func mergeBoolDefault(first, second *bool) *bool {
	if first == nil {
		return second
	}
	if second == nil || *first == *second {
		return first
	}
	return nil
}

func mergeIntDefault(first, second *int64) *int64 {
	if first == nil {
		return second
	}
	if second == nil || *first == *second {
		return first
	}
	return nil
}

func mergeFloatDefault(first, second *float64) *float64 {
	if first == nil {
		return second
	}
	if second == nil || *first == *second {
		return first
	}
	return nil
}

func mergeTimeDefault(first, second *time.Time) *time.Time {
	if first == nil {
		return second
	}
	if second == nil || first.Equal(*second) {
		return first
	}
	return nil
}

func mapValues[T any](values map[string]T) []T {
	result := make([]T, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func mapContactValues(values map[string]domain.ContactPoint) []domain.ContactPoint {
	result := make([]domain.ContactPoint, 0, len(values))
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value.ID]; ok {
			continue
		}
		seen[value.ID] = struct{}{}
		result = append(result, value)
	}
	return result
}

func edgeKey(parentID, childID uuid.UUID) string {
	return parentID.String() + "\x00" + childID.String()
}

func mapEdges(values map[string]domain.ImportCategoryEdge) []domain.ImportCategoryEdge {
	result := make([]domain.ImportCategoryEdge, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func mapOrganizationEdges(values map[string]domain.ImportOrganizationEdge) []domain.ImportOrganizationEdge {
	result := make([]domain.ImportOrganizationEdge, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
