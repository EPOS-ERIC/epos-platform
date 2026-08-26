package clients

import (
	"fmt"
	"slices"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

// Optional scalar, passed straight through
func setIfPresent[T any](v *T, set func(T)) {
	if v != nil {
		set(*v)
	}
}

// Optional value that needs converting first (pointer -> struct conversion)
func setPtrConv[T, K any](v *T, conv func(T) K, set func(K)) {
	if v != nil {
		set(conv(*v))
	}
}

// Non-empty slice that needs converting first
func setSliceConv[T, K any](v []T, conv func([]T) K, set func(K)) {
	if len(v) > 0 {
		set(conv(v))
	}
}

func mapSlice[T, K any](values []T, f func(T) K) []K {
	if values == nil {
		return nil
	}
	result := make([]K, len(values))
	for i, value := range values {
		result[i] = f(value)
	}
	return result
}

func mapSliceErr[T, K any](values []T, f func(T) (K, error)) ([]K, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]K, len(values))
	for i, value := range values {
		mapped, err := f(value)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
	}
	return result, nil
}

func toClientDatasetDraftMetadata(metadata domain.DatasetDraftMetadata) client.DatasetDraftMetadata {
	result := client.DatasetDraftMetadata{ //nolint:exhaustruct_v5 // we set the values manually later
		// TODO: see if we can skip the struct and just use the title as a string
		Title: metadata.Title,
	}

	setIfPresent(metadata.Description, result.SetDescription)
	setIfPresent(metadata.Keywords, result.SetKeywords)
	setIfPresent(metadata.AccrualPeriodicity, result.SetAccrualPeriodicity)
	setIfPresent(metadata.QualityAnnotation, result.SetQualityAnnotation)
	setIfPresent(metadata.AccessRight, result.SetAccessRight)
	setIfPresent(metadata.SourceCreated, result.SetCreated)
	setIfPresent(metadata.SourceUpdated, result.SetUpdated)

	result.SetId(metadata.ID)

	setPtrConv(metadata.SpatialCoverage, toClientSpatialCoverage, result.SetSpatialCoverage)
	setPtrConv(metadata.TemporalCoverage, toClientTemporalCoverage, result.SetTemporalCoverage)

	setSliceConv(metadata.Distributions, toClientDistributionDrafts, result.SetDistributions)
	setSliceConv(metadata.PersistentIdentifiers, toClientIdentifierDrafts, result.SetPersistentIdentifiers)
	setSliceConv(metadata.ContactPoints, toClientResourceReferences, result.SetContactPoints)
	setSliceConv(metadata.Providers, toClientResourceReferences, result.SetProviders)
	setSliceConv(metadata.Categories, toClientResourceReferences, result.SetCategories)

	return result
}

func toClientCreateResourceRequest(resourceType string, metadata domain.DraftMetadata) (client.CreateResourceRequest, error) {
	switch metadata := metadata.(type) {
	case domain.DatasetDraftMetadata:
		return client.CreateDatasetResourceRequestAsCreateResourceRequest(&client.CreateDatasetResourceRequest{
			ResourceType: resourceType,
			Metadata:     toClientDatasetDraftMetadata(metadata),
		}), nil
	case domain.SoftwareApplicationDraftMetadata:
		return client.CreateSoftwareApplicationResourceRequestAsCreateResourceRequest(&client.CreateSoftwareApplicationResourceRequest{
			ResourceType: resourceType,
			Metadata:     toClientSoftwareApplicationDraftMetadata(metadata),
		}), nil
	case domain.SoftwareSourceCodeDraftMetadata:
		return client.CreateSoftwareSourceCodeResourceRequestAsCreateResourceRequest(&client.CreateSoftwareSourceCodeResourceRequest{
			ResourceType: resourceType,
			Metadata:     toClientSoftwareSourceCodeDraftMetadata(metadata),
		}), nil
	default:
		return client.CreateResourceRequest{}, fmt.Errorf("unsupported lifecycle metadata type %T", metadata)
	}
}

func toClientReplaceResourceDraftRequest(input domain.ReplaceResourceDraftInput) (client.ReplaceResourceDraftRequest, error) {
	switch metadata := input.Metadata.(type) {
	case domain.DatasetDraftMetadata:
		return client.ReplaceResourceDraftRequest{ //nolint:exhaustruct_v5 // union request sets one variant
			ReplaceDraftRequest: &client.ReplaceDraftRequest{
				ResourceType: "dataset",
				Revision:     input.Revision,
				Metadata:     toClientDatasetDraftMetadata(metadata),
			},
		}, nil
	case domain.SoftwareApplicationDraftMetadata:
		return client.ReplaceResourceDraftRequest{ //nolint:exhaustruct_v5 // union request sets one variant
			ReplaceSoftwareApplicationDraftRequest: &client.ReplaceSoftwareApplicationDraftRequest{
				ResourceType: "software-application",
				Revision:     input.Revision,
				Metadata:     toClientSoftwareApplicationDraftMetadata(metadata),
			},
		}, nil
	case domain.SoftwareSourceCodeDraftMetadata:
		return client.ReplaceResourceDraftRequest{ //nolint:exhaustruct_v5 // union request sets one variant
			ReplaceSoftwareSourceCodeDraftRequest: &client.ReplaceSoftwareSourceCodeDraftRequest{
				ResourceType: "software-source-code",
				Revision:     input.Revision,
				Metadata:     toClientSoftwareSourceCodeDraftMetadata(metadata),
			},
		}, nil
	default:
		return client.ReplaceResourceDraftRequest{}, fmt.Errorf("unsupported lifecycle metadata type %T", input.Metadata)
	}
}

func toClientSoftwareApplicationDraftMetadata(metadata domain.SoftwareApplicationDraftMetadata) client.SoftwareApplicationDraftMetadata {
	result := client.SoftwareApplicationDraftMetadata{ //nolint:exhaustruct_v5 // optional values are set below
		// TODO: see if we can skip the struct and just use the title as a string
		Name: metadata.Name,
	}

	setIfPresent(metadata.Description, result.SetDescription)
	setIfPresent(metadata.Keywords, result.SetKeywords)
	setIfPresent(metadata.License, result.SetLicense)
	setIfPresent(metadata.Version, result.SetVersion)
	setIfPresent(metadata.DownloadURL, result.SetDownloadUrl)
	setIfPresent(metadata.MainEntityOfPage, result.SetMainEntityOfPage)
	setIfPresent(metadata.Requirements, result.SetRequirements)
	setIfPresent(metadata.OperatingSystem, result.SetOperatingSystem)
	setSliceConv(metadata.Creators, toClientResourceReferences, result.SetCreators)
	setSliceConv(metadata.Identifiers, toClientIdentifierDrafts, result.SetIdentifiers)
	setSliceConv(metadata.Categories, toClientResourceReferences, result.SetCategories)
	setSliceConv(metadata.ContactPoints, toClientResourceReferences, result.SetContactPoints)

	return result
}

func toClientSoftwareSourceCodeDraftMetadata(metadata domain.SoftwareSourceCodeDraftMetadata) client.SoftwareSourceCodeDraftMetadata {
	result := client.SoftwareSourceCodeDraftMetadata{ //nolint:exhaustruct_v5 // optional values are set below
		// TODO: see if we can skip the struct and just use the title as a string
		Name: metadata.Name,
	}

	setIfPresent(metadata.Description, result.SetDescription)
	setIfPresent(metadata.Keywords, result.SetKeywords)
	setIfPresent(metadata.Repository, result.SetRepository)
	setIfPresent(metadata.ProgrammingLanguage, result.SetProgrammingLanguage)
	setIfPresent(metadata.License, result.SetLicense)
	setIfPresent(metadata.SoftwareRequirements, result.SetSoftwareRequirements)
	setIfPresent(metadata.RuntimePlatform, result.SetRuntimePlatform)
	setIfPresent(metadata.MainEntityOfPage, result.SetMainEntityOfPage)
	setIfPresent(metadata.Version, result.SetVersion)
	setSliceConv(metadata.Creators, toClientResourceReferences, result.SetCreators)
	setSliceConv(metadata.Identifiers, toClientIdentifierDrafts, result.SetIdentifiers)
	setSliceConv(metadata.Categories, toClientResourceReferences, result.SetCategories)
	setSliceConv(metadata.ContactPoints, toClientResourceReferences, result.SetContactPoints)

	return result
}

func toClientResourceReferences(values []domain.ResourceReference) []client.CanonicalReference {
	return mapSlice(values, func(value domain.ResourceReference) client.CanonicalReference {
		return client.CanonicalReference{Id: value.ID}
	})
}

func toClientIdentifierDrafts(values []domain.Identifier) []client.IdentifierDraft {
	return mapSlice(values, func(value domain.Identifier) client.IdentifierDraft {
		result := client.IdentifierDraft{ //nolint:exhaustruct_v5 // optional ID is set below
			Type:  value.Type,
			Value: value.Value,
		}
		result.SetId(value.ID)
		return result
	})
}

func toClientSpatialCoverage(value domain.SpatialCoverage) client.Spatial {
	return client.Spatial{
		Point:   toClientCoordinatePtr(value.Point),
		Polygon: mapSlice(value.Polygon, toClientCoordinate),
	}
}

func toClientCoordinatePtr(value *domain.Coordinate) *client.Coordinate {
	if value == nil {
		return nil
	}
	return &client.Coordinate{Latitude: float32(value.Latitude), Longitude: float32(value.Longitude)}
}

func toClientCoordinate(value domain.Coordinate) client.Coordinate {
	return client.Coordinate{Latitude: float32(value.Latitude), Longitude: float32(value.Longitude)}
}

func toClientTemporalCoverage(value domain.TemporalCoverage) client.Temporal {
	result := client.Temporal{} //nolint:exhaustruct_v5 // optional values are set below
	setIfPresent(value.Start, result.SetStart)
	setIfPresent(value.End, result.SetEnd)
	return result
}

func toClientDistributionDrafts(values []domain.DistributionDraft) []client.DistributionDraft {
	return mapSlice(values, toClientDistributionDraft)
}

func toClientDistributionDraft(value domain.DistributionDraft) client.DistributionDraft {
	if value.DownloadableFile != nil {
		result := &client.DownloadableFileDistributionDraft{ //nolint:exhaustruct_v5 // optional values are set below
			Title:            value.Title,
			Description:      new(value.Description),
			Type:             value.Type,
			DownloadableFile: client.DownloadableFile{Url: value.DownloadableFile.URL, Format: value.DownloadableFile.Format},
		}
		result.SetId(value.ID)
		setNullable(&result.License, value.License)
		result.Issued.Set(value.Issued)
		result.Modified.Set(value.Modified)
		result.ByteSize.Set(value.ByteSize)

		return client.DownloadableFileDistributionDraftAsDistributionDraft(result)
	}

	// has to be a webservice if it's not a DownloadableFile

	result := &client.WebServiceDistributionDraft{ //nolint:exhaustruct_v5 // optional values are set below
		Title:       value.Title,
		Description: new(value.Description),
		Type:        value.Type,
		WebService:  toClientWebServiceDraft(*value.WebService),
	}
	result.SetId(value.ID)
	setNullable(&result.License, value.License)
	result.Issued.Set(value.Issued)
	result.Modified.Set(value.Modified)
	result.ByteSize.Set(value.ByteSize)

	return client.WebServiceDistributionDraftAsDistributionDraft(result)
}

func toClientWebServiceDraft(value domain.WebServiceDraft) client.WebServiceDraft {
	result := client.WebServiceDraft{ //nolint:exhaustruct_v5 // optional values are set below
		Name:        value.Name,
		Description: new(value.Description),
		TemplateUrl: value.TemplateURL,
		Format:      value.Format,
	}
	result.SetId(value.ID)
	if value.Provider != nil {
		result.SetProvider(client.CanonicalReference{Id: value.Provider.ID})
	}
	setPtrConv(value.Documentation, toClientDocumentation, result.SetDocumentation)
	setPtrConv(value.SpatialCoverage, toClientSpatialCoverage, result.SetSpatialCoverage)
	setPtrConv(value.TemporalCoverage, toClientTemporalCoverage, result.SetTemporalCoverage)
	setSliceConv(value.ContactPoints, toClientResourceReferences, result.SetContactPoints)

	parameters := make([]client.Parameter, 0, len(value.Parameters))
	for _, parameter := range value.Parameters {
		if mapped, ok := parameter.(client.Parameter); ok {
			parameters = append(parameters, mapped)
		}
	}
	if len(parameters) > 0 {
		result.SetParameters(parameters)
	}

	return result
}

func toClientDocumentation(value domain.Documentation) client.Documentation {
	result := client.Documentation{Uri: value.URI} //nolint:exhaustruct_v5 // nullable fields are set below
	setNullable(&result.Title, value.Title)
	setNullable(&result.Description, value.Description)
	return result
}

func toDomainResourceVersion(record *client.ResourceRecord) (domain.ResourceVersion, error) {
	if record == nil {
		return domain.ResourceVersion{}, fmt.Errorf("decode metadata-lifecycle resource record: response is empty")
	}

	result := domain.ResourceVersion{ //nolint:exhaustruct_v5 // metadata is set from response variant below
		ResourceID: record.ResourceId,
		VersionID:  record.VersionId,
		Revision:   record.Revision,
		Status:     string(record.Status),
		CreatedBy:  record.CreatedBy,
		GroupID:    record.GroupId,
		GroupName:  record.GroupName,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
	}

	switch {
	case record.Metadata.DatasetMetadata != nil:
		metadata := record.Metadata.DatasetMetadata
		result.Metadata = toDomainDatasetResourceMetadata(metadata)
	case record.Metadata.SoftwareApplicationMetadata != nil:
		metadata := record.Metadata.SoftwareApplicationMetadata
		result.Metadata = toDomainSoftwareApplicationResourceMetadata(metadata)
	case record.Metadata.SoftwareSourceCodeMetadata != nil:
		metadata := record.Metadata.SoftwareSourceCodeMetadata
		result.Metadata = toDomainSoftwareSourceCodeResourceMetadata(metadata)
	default:
		return domain.ResourceVersion{}, fmt.Errorf("decode metadata-lifecycle resource metadata: response is empty")
	}

	return result, nil
}

func toDomainDatasetResourceMetadata(value *client.DatasetMetadata) domain.DatasetResourceMetadata {
	return domain.DatasetResourceMetadata{
		ID:                    value.Id,
		Title:                 value.Title,
		Description:           value.Description,
		Keywords:              value.Keywords,
		AccrualPeriodicity:    value.AccrualPeriodicity.Get(),
		QualityAnnotation:     value.QualityAnnotation.Get(),
		AccessRight:           value.AccessRight.Get(),
		Distributions:         toDomainDistributions(value.Distributions),
		SpatialCoverage:       mapClientSpatial(value.SpatialCoverage.Get()),
		TemporalCoverage:      mapClientTemporal(value.TemporalCoverage),
		PersistentIdentifiers: mapSlice(value.PersistentIdentifiers, toDomainIdentifier),
		ContactPoints:         mapSlice(value.ContactPoints, toDomainContactPointReference),
		Providers:             mapSlice(value.Providers, toDomainResourceVersionReference),
		Categories:            mapSlice(value.Categories, toDomainResourceVersionReference),
		SourceCreated:         value.Created.Get(),
		SourceUpdated:         value.Updated.Get(),
	}
}

func toDomainSoftwareApplicationResourceMetadata(value *client.SoftwareApplicationMetadata) domain.SoftwareApplicationResourceMetadata {
	return domain.SoftwareApplicationResourceMetadata{
		ID:               value.Id,
		Name:             value.Name,
		Description:      value.Description,
		MainEntityOfPage: value.MainEntityOfPage.Get(),
		License:          value.License.Get(),
		Version:          value.Version.Get(),
		Keywords:         value.Keywords,
		DownloadURL:      value.DownloadUrl.Get(),
		Requirements:     value.Requirements.Get(),
		OperatingSystem:  value.OperatingSystem.Get(),
		Creators:         mapSlice(value.Creators, toDomainResourceVersionReference),
		Identifiers:      mapSlice(value.Identifiers, toDomainIdentifier),
		ContactPoints:    mapSlice(value.ContactPoints, toDomainContactPointReference),
		Categories:       mapSlice(value.Categories, toDomainResourceVersionReference),
	}
}

func toDomainSoftwareSourceCodeResourceMetadata(value *client.SoftwareSourceCodeMetadata) domain.SoftwareSourceCodeResourceMetadata {
	return domain.SoftwareSourceCodeResourceMetadata{
		ID:                   value.Id,
		Name:                 value.Name,
		Description:          value.Description,
		ProgrammingLanguage:  value.ProgrammingLanguage.Get(),
		Keywords:             value.Keywords,
		Repository:           value.Repository.Get(),
		License:              value.License.Get(),
		SoftwareRequirements: value.SoftwareRequirements.Get(),
		RuntimePlatform:      value.RuntimePlatform.Get(),
		MainEntityOfPage:     value.MainEntityOfPage.Get(),
		Version:              value.Version.Get(),
		Creators:             mapSlice(value.Creators, toDomainResourceVersionReference),
		Identifiers:          mapSlice(value.Identifiers, toDomainIdentifier),
		ContactPoints:        mapSlice(value.ContactPoints, toDomainContactPointReference),
		Categories:           mapSlice(value.Categories, toDomainResourceVersionReference),
	}
}

func toDomainResourceVersionReference(value client.ResourceReference) domain.ResourceVersionReference {
	return domain.ResourceVersionReference{
		ID:   value.Id,
		Name: value.Name,
	}
}

func toDomainContactPointReference(value client.ContactPointReference) domain.ContactPointReference {
	// TODO: see if we can skip the struct and just use the id directly
	return domain.ContactPointReference{
		ID: value.Id,
	}
}

func mapClientSpatial(value *client.Spatial) *domain.SpatialCoverage {
	if value == nil {
		return nil
	}
	return new(domain.SpatialCoverage{
		Point:   toDomainCoordinatePtr(value.Point),
		Polygon: mapSlice(value.Polygon, toDomainCoordinate),
	})
}

func toDomainCoordinatePtr(value *client.Coordinate) *domain.Coordinate {
	if value == nil {
		return nil
	}
	return new(domain.Coordinate{
		Latitude:  float64(value.Latitude),
		Longitude: float64(value.Longitude),
	})
}

func toDomainCoordinate(value client.Coordinate) domain.Coordinate {
	return domain.Coordinate{
		Latitude:  float64(value.Latitude),
		Longitude: float64(value.Longitude),
	}
}

func mapClientTemporal(value *client.Temporal) *domain.TemporalCoverage {
	if value == nil {
		return nil
	}
	return new(domain.TemporalCoverage{Start: value.Start.Get(), End: value.End.Get()})
}

func toDomainDistributions(values []client.Distribution) []domain.Distribution {
	if values == nil {
		return nil
	}
	result := make([]domain.Distribution, 0, len(values))
	for _, value := range values {
		switch {
		case value.WebServiceDistribution != nil:
			v := value.WebServiceDistribution
			result = append(result, domain.WebServiceDistribution{
				ID:          v.Id,
				Title:       v.Title,
				Description: v.Description,
				License:     v.License.Get(),
				Type:        v.Type,
				Issued:      v.Issued.Get(),
				Modified:    v.Modified.Get(),
				ByteSize:    v.ByteSize.Get(),
				WebService:  toDomainWebService(v.WebService),
			})
		case value.DownloadableFileDistribution != nil:
			v := value.DownloadableFileDistribution
			result = append(result, domain.DownloadableFileDistribution{
				ID:          v.Id,
				Title:       v.Title,
				Description: v.Description,
				License:     v.License.Get(),
				Type:        v.Type,
				Issued:      v.Issued.Get(),
				Modified:    v.Modified.Get(),
				ByteSize:    v.ByteSize.Get(),
				DownloadableFile: domain.DownloadableFile{
					URL:    v.DownloadableFile.Url,
					Format: v.DownloadableFile.Format,
				},
			})
		}
	}
	return result
}

func toDomainWebService(value client.WebService) domain.WebService {
	return domain.WebService{
		ID:               value.Id,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateUrl,
		Parameters:       toDomainParameters(value.Parameters),
		Format:           value.Format,
		Provider:         mapClientResourceReference(value.Provider),
		Documentation:    mapClientDocumentation(value.Documentation),
		SpatialCoverage:  mapClientSpatial(value.SpatialCoverage.Get()),
		TemporalCoverage: mapClientTemporal(value.TemporalCoverage),
		ContactPoints:    mapSlice(value.ContactPoints, toDomainContactPointReference),
	}
}

func mapClientResourceReference(value *client.ResourceReference) *domain.ResourceVersionReference {
	if value == nil {
		return nil
	}
	return new(toDomainResourceVersionReference(*value))
}

func mapClientDocumentation(value *client.Documentation) *domain.Documentation {
	if value == nil {
		return nil
	}
	return new(domain.Documentation{
		Title:       value.Title.Get(),
		Description: value.Description.Get(),
		URI:         value.Uri,
	})
}

func toDomainParameters(values []client.Parameter) []domain.Parameter {
	if values == nil {
		return nil
	}
	result := make([]domain.Parameter, 0, len(values))
	for _, value := range values {
		switch {
		case value.StringParameter != nil:
			v := value.StringParameter
			result = append(result, domain.StringParameter{
				Metadata:      toDomainParameterMetadata(&v.Metadata),
				Default:       v.Default.Get(),
				AllowedValues: slices.Clone(v.AllowedValues),
				Pattern:       v.Pattern,
			})
		case value.BooleanParameter != nil:
			v := value.BooleanParameter
			result = append(result, domain.BooleanParameter{
				Metadata: toDomainParameterMetadata(&v.Metadata),
				Default:  v.Default.Get(),
			})
		case value.IntegerParameter != nil:
			v := value.IntegerParameter
			result = append(result, domain.IntegerParameter{
				Metadata:      toDomainParameterMetadata(&v.Metadata),
				Minimum:       v.Minimum.Get(),
				Maximum:       v.Maximum.Get(),
				Default:       v.Default.Get(),
				AllowedValues: slices.Clone(v.AllowedValues),
			})
		case value.FloatParameter != nil:
			v := value.FloatParameter
			result = append(result, domain.FloatParameter{
				Metadata:      toDomainParameterMetadata(&v.Metadata),
				Minimum:       v.Minimum.Get(),
				Maximum:       v.Maximum.Get(),
				Default:       v.Default.Get(),
				AllowedValues: slices.Clone(v.AllowedValues),
			})
		case value.DateTimeParameter != nil:
			v := value.DateTimeParameter
			result = append(result, domain.DateTimeParameter{
				Metadata: toDomainParameterMetadata(&v.Metadata),
				Minimum:  v.Minimum.Get(),
				Maximum:  v.Maximum.Get(),
				Default:  v.Default.Get(),
				Pattern:  v.Pattern,
			})
		}
	}
	return result
}

type parameterMetadataAccessor interface {
	GetId() string
	GetKind() string
	GetVariable() string
	GetProperty() string
	GetLabel() string
	GetRequired() bool
	GetMultipleValues() bool
	GetReadOnly() bool
}

func toDomainParameterMetadata(value parameterMetadataAccessor) domain.ParameterMetadata {
	return domain.ParameterMetadata{
		ID:             value.GetId(),
		Kind:           value.GetKind(),
		Variable:       value.GetVariable(),
		Property:       value.GetProperty(),
		Label:          value.GetLabel(),
		Required:       value.GetRequired(),
		MultipleValues: value.GetMultipleValues(),
		ReadOnly:       value.GetReadOnly(),
	}
}

func toDomainContactPoint(value client.ContactPoint) domain.ContactPoint {
	return domain.ContactPoint{
		ID:    value.Id,
		Email: value.Email,
		Role:  value.Role,
	}
}

func toDomainCanonicalOrganization(value client.CanonicalOrganization) domain.CanonicalOrganization {
	return domain.CanonicalOrganization{
		ID:              value.Id,
		Name:            value.Name,
		Address:         toDomainOrganizationAddress(value.Address),
		LogoURL:         value.LogoUrl.Get(),
		PageURL:         value.PageUrl.Get(),
		Email:           value.Email.Get(),
		LEICode:         value.LeiCode.Get(),
		Parents:         slices.Clone(value.Parents),
		ContactPointIDs: slices.Clone(value.ContactPointIds),
		Identifiers:     mapSlice(value.Identifiers, toDomainIdentifier),
	}
}

func toDomainIdentifier(value client.Identifier) domain.Identifier {
	return domain.Identifier{
		ID:    value.Id,
		Type:  value.Type,
		Value: value.Value,
	}
}

func toDomainCategory(value client.CanonicalCategory) domain.Category {
	return domain.Category{
		ID:               value.Id,
		GroupID:          value.GroupId,
		ParentCategoryID: value.ParentCategoryId.Get(),
		Name:             value.Name,
	}
}

func toClientCreateContactPointRequest(value domain.CreateContactPointInput) client.CreateContactPointRequest {
	return client.CreateContactPointRequest{
		Email: value.Email,
		Role:  value.Role,
	}
}

func toClientReplaceContactPointRequest(value domain.ReplaceContactPointInput) client.ReplaceContactPointRequest {
	return client.ReplaceContactPointRequest{
		Email: value.Email,
		Role:  value.Role,
	}
}

func toClientOrganizationIdentifierInput(value domain.OrganizationIdentifierInput) client.CreateOrganizationIdentifierRequest {
	return client.CreateOrganizationIdentifierRequest{
		Type:  value.Type,
		Value: value.Value,
	}
}

func toClientCreateCanonicalOrganizationRequest(value domain.CreateOrganizationInput) client.CreateCanonicalOrganizationRequest {
	result := client.CreateCanonicalOrganizationRequest{ //nolint:exhaustruct_v5 // optional values are set below
		Name:            value.Name,
		Identifiers:     mapSlice(value.Identifiers, toClientOrganizationIdentifierInput),
		Parents:         value.Parents,
		ContactPointIds: value.ContactPointIDs,
	}
	setPtrConv(value.Address, toClientCreateOrganizationAddress, func(address *client.CreateCanonicalOrganizationRequestAddress) {
		result.Address = address
	})
	setNullable(&result.LogoUrl, value.LogoURL)
	setNullable(&result.PageUrl, value.PageURL)
	setNullable(&result.Email, value.Email)
	setNullable(&result.LeiCode, value.LEICode)
	return result
}

func toClientReplaceCanonicalOrganizationRequest(value domain.ReplaceOrganizationInput) client.ReplaceCanonicalOrganizationRequest {
	result := client.ReplaceCanonicalOrganizationRequest{ //nolint:exhaustruct_v5 // optional values are set below
		Name:            value.Name,
		Address:         toClientOrganizationAddress(value.Address),
		Identifiers:     mapSlice(value.Identifiers, toClientOrganizationIdentifierInput),
		Parents:         value.Parents,
		ContactPointIds: value.ContactPointIDs,
	}
	setNullable(&result.LogoUrl, value.LogoURL)
	setNullable(&result.PageUrl, value.PageURL)
	setNullable(&result.Email, value.Email)
	setNullable(&result.LeiCode, value.LEICode)
	return result
}

func toClientCreateOrganizationAddress(value domain.OrganizationAddress) *client.CreateCanonicalOrganizationRequestAddress {
	result := &client.CreateCanonicalOrganizationRequestAddress{} //nolint:exhaustruct_v5 // optional values are set below
	setNullable(&result.Street, value.Street)
	setNullable(&result.Locality, value.Locality)
	setNullable(&result.PostalCode, value.PostalCode)
	setNullable(&result.Country, value.Country)
	return result
}

func toClientOrganizationAddress(value domain.OrganizationAddress) client.CanonicalOrganizationAddress {
	result := client.CanonicalOrganizationAddress{} //nolint:exhaustruct_v5 // optional values are set below
	setNullable(&result.Street, value.Street)
	setNullable(&result.Locality, value.Locality)
	setNullable(&result.PostalCode, value.PostalCode)
	setNullable(&result.Country, value.Country)
	return result
}

func toDomainOrganizationAddress(value client.CanonicalOrganizationAddress) domain.OrganizationAddress {
	return domain.OrganizationAddress{
		Street:     value.Street.Get(),
		Locality:   value.Locality.Get(),
		PostalCode: value.PostalCode.Get(),
		Country:    value.Country.Get(),
	}
}

func setNullable(value *client.NullableString, input *string) {
	if input == nil {
		value.Set(nil)
		return
	}
	value.Set(input)
}

func toClientCreateGroupRequest(value domain.CreateGroupInput) client.CreateGroupRequest {
	result := client.CreateGroupRequest{ //nolint:exhaustruct_v5 // optional values are set below
		Name:     value.Name,
		Position: value.Position,
		Concept:  value.Concept,
	}
	setNullable(&result.Description, value.Description)
	setNullable(&result.Label, value.Label)
	setNullable(&result.Logo, value.Logo)
	setNullable(&result.Page, value.Page)
	setNullable(&result.Color, value.Color)
	return result
}

func toClientReplaceGroupRequest(value domain.ReplaceGroupInput) client.ReplaceGroupRequest {
	result := client.ReplaceGroupRequest{ //nolint:exhaustruct_v5 // optional values are set below
		Name:     value.Name,
		Position: value.Position,
		Concept:  value.Concept,
	}
	setNullable(&result.Description, value.Description)
	setNullable(&result.Label, value.Label)
	setNullable(&result.Logo, value.Logo)
	setNullable(&result.Page, value.Page)
	setNullable(&result.Color, value.Color)
	return result
}

func toClientCreateCategoryRequest(value domain.CreateCategoryInput) client.CreateCategoryRequest {
	return client.CreateCategoryRequest{
		Name:             value.Name,
		ParentCategoryId: value.ParentCategoryID,
	}
}

func toClientReplaceCategoryRequest(value domain.ReplaceCategoryInput) client.ReplaceCategoryRequest {
	result := client.ReplaceCategoryRequest{ //nolint:exhaustruct_v5 // optional parent ID is set below
		Name: value.Name,
	}
	setNullable(&result.ParentCategoryId, value.ParentCategoryID)
	return result
}

func toClientResourceTypes(values []string) []client.ResourceType {
	return mapSlice(values, func(value string) client.ResourceType { return client.ResourceType(value) })
}

func toDomainDiscoveryItem(value client.DiscoveryItem) domain.DiscoveryItem {
	return domain.DiscoveryItem{
		Type:          value.Type,
		ResourceID:    value.ResourceId,
		VersionID:     value.VersionId,
		Name:          value.Name,
		Status:        string(value.Status),
		Group:         toDomainDiscoveryReference(value.Group),
		Providers:     mapSlice(value.Providers, toDomainDiscoveryReference),
		Creators:      mapSlice(value.Creators, toDomainDiscoveryReference),
		Categories:    mapSlice(value.Categories, toDomainDiscoveryReference),
		Distributions: mapSlice(value.Distributions, toDomainDiscoveryReference),
	}
}

func toDomainDiscoveryReference(value client.DiscoveryReference) domain.DiscoveryReference {
	return domain.DiscoveryReference{
		ID:   value.Id,
		Name: value.Name,
	}
}

func toDomainUser(value *client.User) (domain.User, error) {
	if value == nil {
		return domain.User{}, fmt.Errorf("decode metadata-lifecycle user: response is empty")
	}

	memberships := mapSlice(value.Memberships, func(membership client.UserMembership) domain.Membership {
		return domain.Membership{
			Group: domain.Group{ //nolint:exhaustruct_v5 // membership reference only carries ID and name
				ID:   membership.Group.Id,
				Name: membership.Group.Name,
			},
			Roles:        slices.Clone(membership.Roles),
			Status:       membership.Status,
			Permissions:  slices.Clone(membership.Permissions),
			RoleRequests: mapSlice(membership.RoleRequests, toDomainRoleRequest),
			CreatedAt:    membership.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:    membership.UpdatedAt.Format(time.RFC3339Nano),
		}
	})

	return domain.User{
		ID:                value.Id,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            value.Status,
		GlobalRoles:       slices.Clone(value.GlobalRoles),
		GlobalPermissions: slices.Clone(value.GlobalPermissions),
		CreatedAt:         value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         value.UpdatedAt.Format(time.RFC3339Nano),
		Memberships:       memberships,
	}, nil
}

func toDomainRoleRequest(value client.RoleRequest) domain.RoleRequest {
	return domain.RoleRequest{
		ID:          value.Id,
		Role:        value.Role,
		Status:      value.Status,
		RequestedAt: value.RequestedAt,
	}
}

func toDomainGroup(value *client.Group) (domain.Group, error) {
	if value == nil {
		return domain.Group{}, fmt.Errorf("decode metadata-lifecycle group: response is empty")
	}

	return domain.Group{
		ID:          value.Id,
		Name:        value.Name,
		Description: value.Description.Get(),
		Label:       value.Label.Get(),
		Logo:        value.Logo.Get(),
		Page:        value.Page.Get(),
		Color:       value.Color.Get(),
		Position:    value.Position,
		Concept:     value.Concept,
	}, nil
}

func toDomainGroups(values []client.Group) ([]domain.Group, error) {
	return mapSliceErr(values, func(value client.Group) (domain.Group, error) {
		return toDomainGroup(new(value))
	})
}

func toDomainGroupMembership(value *client.GroupMembership) (domain.GroupMembership, error) {
	if value == nil {
		return domain.GroupMembership{}, fmt.Errorf("decode metadata-lifecycle membership: response is empty")
	}

	return domain.GroupMembership{
		User: domain.MembershipUser{
			ID:         value.User.Id,
			GivenName:  value.User.GivenName,
			FamilyName: value.User.FamilyName,
			Email:      value.User.Email,
		},
		Status:       value.Status,
		Roles:        slices.Clone(value.Roles),
		Permissions:  slices.Clone(value.Permissions),
		RoleRequests: mapSlice(value.RoleRequests, toDomainRoleRequest),
		CreatedAt:    value.CreatedAt,
		UpdatedAt:    value.UpdatedAt,
	}, nil
}

func toDomainGroupMembershipValue(value client.GroupMembership) (domain.GroupMembership, error) {
	return toDomainGroupMembership(new(value))
}

func toDomainUserSummary(value client.UserSummary) domain.UserSummary {
	return domain.UserSummary{ //nolint:exhaustruct_v5 // summary response has no timestamps
		ID:                value.Id,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            value.Status,
		GlobalRoles:       slices.Clone(value.GlobalRoles),
		GlobalPermissions: slices.Clone(value.GlobalPermissions),
	}
}

func toDomainImportDiagnostic(value client.ImportDiagnostic) domain.ImportDiagnostic {
	return domain.ImportDiagnostic(value)
}

func toDomainImportResult(value client.ImportResult) domain.ImportResult {
	return domain.ImportResult{
		Groups:               value.Groups,
		Categories:           value.Categories,
		Organizations:        value.Organizations,
		ContactPoints:        value.ContactPoints,
		Datasets:             value.Datasets,
		SoftwareApplications: value.SoftwareApplications,
		SoftwareSourceCodes:  value.SoftwareSourceCodes,
		Warnings:             mapSlice(value.Warnings, toDomainImportDiagnostic),
	}
}

func toClientRejectRequest(value domain.RejectResourceInput) client.RejectRequest {
	return client.RejectRequest{
		Note: value.Note,
	}
}

func toClientCopyResourceRequest(groupID string, value domain.CopyResourceInput) client.CopyResourceRequest {
	// TODO: see if we can skip the struct and just use the id directly
	return client.CopyResourceRequest{
		TargetGroupId: groupID,
	}
}

func toClientCreateMembershipRequest(roles []string) client.CreateMembershipRequest {
	return client.CreateMembershipRequest{
		Roles: roles,
	}
}

func toClientMembershipDecisionRequest(status string) client.MembershipDecisionRequest {
	return client.MembershipDecisionRequest{
		Status: status,
	}
}

func toClientCreateRoleRequest(role string) client.CreateRoleRequest {
	return client.CreateRoleRequest{
		Role: role,
	}
}

func toClientRoleRequestDecisionRequest(status string) client.RoleRequestDecisionRequest {
	return client.RoleRequestDecisionRequest{
		Status: status,
	}
}

func toClientBlockUserRequest(reason string) client.BlockUserRequest {
	return client.BlockUserRequest{
		Reason: reason,
	}
}

func toClientCreateUserRequest(identity ActorIdentity) client.CreateUserRequest {
	return client.CreateUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	}
}

func toDomainUserSummaries(values []client.UserSummary) []domain.UserSummary {
	return mapSlice(values, toDomainUserSummary)
}
