package grpcapi

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func datasetDraftInput(value *metadatalifecyclev1.DatasetDraftMetadata) (domain.DatasetInput, error) {
	contactPoints, err := resourceParseUUIDs(value.GetContactPoints(), "dataset contact point")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	providers, err := resourceParseUUIDs(value.GetProviders(), "dataset provider")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "dataset category")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	temporal, err := temporalInput(value.GetTemporalCoverage())
	if err != nil {
		return domain.DatasetInput{}, err
	}
	spatial, err := spatialInput(value.SpatialCoverage)
	if err != nil {
		return domain.DatasetInput{}, err
	}
	distributions := make([]domain.DistributionInput, 0, len(value.GetDistributions()))
	for _, item := range value.GetDistributions() {
		mapped, err := distributionDraftInput(item)
		if err != nil {
			return domain.DatasetInput{}, err
		}
		distributions = append(distributions, mapped)
	}
	identifiers := make([]domain.IdentifierInput, 0, len(value.GetPersistentIdentifiers()))
	for _, item := range value.GetPersistentIdentifiers() {
		if item == nil {
			return domain.DatasetInput{}, status.Error(codes.InvalidArgument, "identifier is required") //nolint:wrapcheck
		}
		identifiers = append(identifiers, domain.IdentifierInput{Type: domain.IdentifierType(item.GetType()), Value: item.GetValue()})
	}
	created, err := timestampInput(value.GetCreated())
	if err != nil {
		return domain.DatasetInput{}, err
	}
	updated, err := timestampInput(value.GetUpdated())
	if err != nil {
		return domain.DatasetInput{}, err
	}
	return domain.DatasetInput{Title: value.GetTitle(), Description: value.GetDescription(), Keywords: value.GetKeywords(), AccrualPeriodicity: value.AccrualPeriodicity, QualityAnnotation: value.QualityAnnotation, AccessRight: value.AccessRight, Distributions: distributions, SpatialCoverage: spatial, TemporalCoverage: temporal, PersistentIdentifiers: identifiers, ContactPointIDs: contactPoints, ProviderIDs: providers, CategoryIDs: categories, SourceCreated: created, SourceUpdated: updated}, nil
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func softwareApplicationDraftInput(value *metadatalifecyclev1.SoftwareApplicationDraftMetadata) (domain.SoftwareApplicationInput, error) {
	creators, err := resourceParseUUIDs(value.GetCreators(), "software application creator")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "software application category")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	contacts, err := resourceParseUUIDs(value.GetContactPoints(), "software application contact point")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	identifiers, err := draftIdentifierInputs(value.GetIdentifiers())
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	return domain.SoftwareApplicationInput{Name: value.GetName(), Description: value.GetDescription(), MainEntityOfPage: value.MainEntityOfPage, License: value.License, Version: value.Version, Keywords: value.GetKeywords(), DownloadURL: value.DownloadUrl, Requirements: value.Requirements, OperatingSystem: value.OperatingSystem, CreatorIDs: creators, ContactPointIDs: contacts, CategoryIDs: categories, Identifiers: identifiers}, nil
}

func softwareSourceCodeDraftInput(value *metadatalifecyclev1.SoftwareSourceCodeDraftMetadata) (domain.SoftwareSourceCodeInput, error) {
	creators, err := resourceParseUUIDs(value.GetCreators(), "software source code creator")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "software source code category")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	contacts, err := resourceParseUUIDs(value.GetContactPoints(), "software source code contact point")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	identifiers, err := draftIdentifierInputs(value.GetIdentifiers())
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	return domain.SoftwareSourceCodeInput{Name: value.GetName(), Description: value.GetDescription(), ProgrammingLanguage: value.ProgrammingLanguage, Keywords: value.GetKeywords(), Repository: value.Repository, License: value.License, SoftwareRequirements: value.SoftwareRequirements, RuntimePlatform: value.RuntimePlatform, MainEntityOfPage: value.MainEntityOfPage, Version: value.Version, CreatorIDs: creators, ContactPointIDs: contacts, CategoryIDs: categories, Identifiers: identifiers}, nil
}

func draftIdentifierInputs(values []*metadatalifecyclev1.IdentifierDraft) ([]domain.IdentifierInput, error) {
	result := make([]domain.IdentifierInput, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, status.Error(codes.InvalidArgument, "identifier is required") //nolint:wrapcheck
		}
		result = append(result, domain.IdentifierInput{Type: domain.IdentifierType(value.GetType()), Value: value.GetValue()})
	}
	return result, nil
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func distributionDraftInput(value *metadatalifecyclev1.DistributionDraft) (domain.DistributionInput, error) {
	if value == nil {
		return domain.DistributionInput{}, status.Error(codes.InvalidArgument, "distribution is required") //nolint:wrapcheck
	}
	issued, err := timestampInput(value.GetIssued())
	if err != nil {
		return domain.DistributionInput{}, err
	}
	modified, err := timestampInput(value.GetModified())
	if err != nil {
		return domain.DistributionInput{}, err
	}
	result := domain.DistributionInput{Title: value.GetTitle(), Description: value.GetDescription(), License: value.License, Type: domain.DistributionType(value.GetType()), Issued: issued, Modified: modified, ByteSize: value.ByteSize}
	switch delivery := value.GetDelivery().(type) {
	case *metadatalifecyclev1.DistributionDraft_WebService:
		mapped, err := webServiceDraftInput(delivery.WebService)
		if err != nil {
			return domain.DistributionInput{}, err
		}
		result.WebService = &mapped
	case *metadatalifecyclev1.DistributionDraft_DownloadableFile:
		if delivery.DownloadableFile == nil {
			return domain.DistributionInput{}, status.Error(codes.InvalidArgument, "downloadable file is required")
		}
		result.DownloadableFile = &domain.DownloadableFile{URL: delivery.DownloadableFile.GetUrl(), Format: delivery.DownloadableFile.GetFormat()}
	default:
		return domain.DistributionInput{}, status.Error(codes.InvalidArgument, "distribution delivery method is required")
	}
	return result, nil
}

func webServiceDraftInput(value *metadatalifecyclev1.WebServiceDraft) (domain.WebServiceInput, error) {
	if value == nil {
		return domain.WebServiceInput{}, status.Error(codes.InvalidArgument, "web service is required")
	}
	contacts, err := resourceParseUUIDs(value.GetContactPoints(), "web service contact point")
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	var provider *uuid.UUID
	if value.Provider != nil {
		id, err := parseUUID(value.GetProvider(), "web service provider")
		if err != nil {
			return domain.WebServiceInput{}, err
		}
		provider = &id
	}
	spatial, err := spatialInput(value.SpatialCoverage)
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	temporal, err := temporalInput(value.GetTemporalCoverage())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	documentation, err := documentationInput(value.GetDocumentation())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	parameters := make([]domain.Parameter, 0, len(value.GetParameters()))
	for _, item := range value.GetParameters() {
		mapped, err := parameterDraftInput(item)
		if err != nil {
			return domain.WebServiceInput{}, err
		}
		parameters = append(parameters, mapped)
	}
	return domain.WebServiceInput{Name: value.GetName(), Description: value.GetDescription(), TemplateURL: value.GetTemplateUrl(), Parameters: parameters, Format: value.GetFormat(), ProviderID: provider, Documentation: documentation, SpatialCoverage: spatial, TemporalCoverage: temporal, ContactPointIDs: contacts}, nil
}

func parameterDraftInput(value *metadatalifecyclev1.ParameterDraft) (domain.Parameter, error) {
	if value == nil || value.GetMetadata() == nil {
		return nil, status.Error(codes.InvalidArgument, "parameter metadata is required")
	}
	m := value.GetMetadata()
	metadata := domain.ParameterMetadata{Variable: m.GetVariable(), Property: m.GetProperty(), Label: m.GetLabel(), Required: m.GetRequired(), MultipleValues: m.GetMultipleValues(), ReadOnly: m.GetReadOnly()}
	switch v := value.GetValue().(type) {
	case *metadatalifecyclev1.ParameterDraft_String_:
		if v.String_ == nil {
			return nil, status.Error(codes.InvalidArgument, "string parameter is required")
		}
		return domain.StringParameter{Metadata: metadata, Default: v.String_.Default, AllowedValues: v.String_.GetAllowedValues(), Pattern: v.String_.GetPattern()}, nil
	case *metadatalifecyclev1.ParameterDraft_Boolean:
		if v.Boolean == nil {
			return nil, status.Error(codes.InvalidArgument, "boolean parameter is required")
		}
		return domain.BooleanParameter{Metadata: metadata, Default: v.Boolean.Default}, nil
	case *metadatalifecyclev1.ParameterDraft_Integer:
		if v.Integer == nil {
			return nil, status.Error(codes.InvalidArgument, "integer parameter is required")
		}
		return domain.IntegerParameter{Metadata: metadata, Minimum: v.Integer.Minimum, Maximum: v.Integer.Maximum, Default: v.Integer.Default, AllowedValues: v.Integer.GetAllowedValues()}, nil
	case *metadatalifecyclev1.ParameterDraft_Float:
		if v.Float == nil {
			return nil, status.Error(codes.InvalidArgument, "float parameter is required")
		}
		return domain.FloatParameter{Metadata: metadata, Minimum: v.Float.Minimum, Maximum: v.Float.Maximum, Default: v.Float.Default, AllowedValues: v.Float.GetAllowedValues()}, nil
	case *metadatalifecyclev1.ParameterDraft_DateTime:
		if v.DateTime == nil {
			return nil, status.Error(codes.InvalidArgument, "date-time parameter is required")
		}
		minimum, err := timestampInput(v.DateTime.GetMinimum())
		if err != nil {
			return nil, err
		}
		maximum, err := timestampInput(v.DateTime.GetMaximum())
		if err != nil {
			return nil, err
		}
		def, err := timestampInput(v.DateTime.GetDefault())
		if err != nil {
			return nil, err
		}
		return domain.DateTimeParameter{Metadata: metadata, Minimum: minimum, Maximum: maximum, Default: def, Pattern: v.DateTime.GetPattern()}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "parameter value is required")
	}
}
