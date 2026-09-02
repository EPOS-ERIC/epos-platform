package dataset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
)

func (p *parameterRequest) UnmarshalJSON(data []byte) error {
	var header struct {
		Kind     domain.ParameterKind `json:"kind"`
		Metadata struct {
			Kind domain.ParameterKind `json:"kind"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return fmt.Errorf("decode parameter metadata: %w", err)
	}

	switch header.Metadata.Kind {
	case domain.ParameterKindString:
		var value stringParameterDraft
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = domain.StringParameter{
			Metadata:      value.Metadata.domain(),
			Default:       value.Default,
			AllowedValues: value.AllowedValues,
			Pattern:       value.Pattern,
		}
	case domain.ParameterKindBoolean:
		var value booleanParameterDraft
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = domain.BooleanParameter{
			Metadata: value.Metadata.domain(),
			Default:  value.Default,
		}
	case domain.ParameterKindInteger:
		var value integerParameterDraft
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = domain.IntegerParameter{
			Metadata:      value.Metadata.domain(),
			Minimum:       value.Minimum,
			Maximum:       value.Maximum,
			Default:       value.Default,
			AllowedValues: value.AllowedValues,
		}
	case domain.ParameterKindFloat:
		var value floatParameterDraft
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = domain.FloatParameter{
			Metadata:      value.Metadata.domain(),
			Minimum:       value.Minimum,
			Maximum:       value.Maximum,
			Default:       value.Default,
			AllowedValues: value.AllowedValues,
		}
	case domain.ParameterKindDateTime:
		var value dateTimeParameterDraft
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = domain.DateTimeParameter{
			Metadata: value.Metadata.domain(),
			Minimum:  value.Minimum,
			Maximum:  value.Maximum,
			Default:  value.Default,
			Pattern:  value.Pattern,
		}
	default:
		return fmt.Errorf("unsupported parameter kind %q", header.Metadata.Kind)
	}
	if header.Kind != header.Metadata.Kind || p.value.Kind() != header.Kind || p.value.GetMetadata().Kind != header.Kind {
		return fmt.Errorf("parameter kind mismatch: kind=%q metadata=%q value=%q", header.Kind, header.Metadata.Kind, p.value.Kind())
	}
	return nil
}

func decodeStrictBytes(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

func (p parameterDraftMetadata) domain() domain.ParameterMetadata {
	return domain.ParameterMetadata{
		ID:             uuid.Nil,
		Kind:           p.Kind,
		Variable:       p.Variable,
		Property:       p.Property,
		Label:          p.Label,
		Required:       p.Required,
		MultipleValues: p.MultipleValues,
		ReadOnly:       p.ReadOnly,
	}
}

func (r datasetDraftRequest) domain() (domain.DatasetInput, error) {
	spatialCoverage, err := spatialGeometry(r.SpatialCoverage)
	if err != nil {
		return domain.DatasetInput{}, err
	}
	result := domain.DatasetInput{
		Title:                 r.Title,
		Description:           r.Description,
		Keywords:              r.Keywords,
		AccrualPeriodicity:    r.AccrualPeriodicity,
		QualityAnnotation:     r.QualityAnnotation,
		AccessRight:           r.AccessRight,
		Distributions:         make([]domain.DistributionInput, 0, len(r.Distributions)),
		SpatialCoverage:       spatialCoverage,
		TemporalCoverage:      temporalDomain(r.TemporalCoverage),
		PersistentIdentifiers: make([]domain.IdentifierInput, 0, len(r.PersistentIdentifiers)),
		ContactPointIDs:       r.ContactPoints,
		ProviderIDs:           r.Providers,
		CategoryIDs:           r.Categories,
		SourceCreated:         r.Created,
		SourceUpdated:         r.Updated,
	}
	for _, value := range r.Distributions {
		distribution, err := value.domain()
		if err != nil {
			return domain.DatasetInput{}, err
		}
		result.Distributions = append(result.Distributions, distribution)
	}
	for _, value := range r.PersistentIdentifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, value.domain())
	}
	return result, nil
}

func (r distributionRequest) domain() (domain.DistributionInput, error) {
	result := domain.DistributionInput{
		Title:            r.Title,
		Description:      r.Description,
		License:          r.License,
		Type:             r.Type,
		Issued:           r.Issued,
		Modified:         r.Modified,
		ByteSize:         r.ByteSize,
		WebService:       nil,
		DownloadableFile: nil,
	}
	if r.WebService != nil {
		value, err := r.WebService.domain()
		if err != nil {
			return domain.DistributionInput{}, err
		}
		result.WebService = &value
	}
	if r.DownloadableFile != nil {
		result.DownloadableFile = &domain.DownloadableFile{
			URL:    r.DownloadableFile.URL,
			Format: r.DownloadableFile.Format,
		}
	}
	return result, nil
}

func (r webServiceRequest) domain() (domain.WebServiceInput, error) {
	spatialCoverage, err := spatialGeometry(r.SpatialCoverage)
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	result := domain.WebServiceInput{
		Name:             r.Name,
		Description:      r.Description,
		TemplateURL:      r.TemplateURL,
		Parameters:       make([]domain.Parameter, 0, len(r.Parameters)),
		Format:           r.Format,
		ProviderID:       r.Provider,
		Documentation:    nil,
		SpatialCoverage:  spatialCoverage,
		TemporalCoverage: temporalDomain(r.TemporalCoverage),
		ContactPointIDs:  r.ContactPoints,
	}
	for _, value := range r.Parameters {
		result.Parameters = append(result.Parameters, value.value)
	}
	if r.Documentation != nil {
		result.Documentation = &domain.Documentation{
			Title:       r.Documentation.Title,
			Description: r.Documentation.Description,
			URI:         r.Documentation.URI,
		}
	}
	return result, nil
}

func spatialGeometry(value *string) (geom.T, error) {
	if value == nil {
		return nil, nil
	}
	geometry, err := wkt.Unmarshal(strings.TrimSpace(*value))
	if err != nil {
		return nil, fmt.Errorf("invalid spatial coverage WKT: %w", err)
	}
	geometry, err = geom.SetSRID(geometry, 4326)
	if err != nil {
		return nil, fmt.Errorf("set spatial coverage SRID: %w", err)
	}
	return geometry, nil
}

func spatialResponse(value geom.T) *string {
	if value == nil {
		return nil
	}
	wktValue, err := wkt.Marshal(value)
	if err != nil {
		return nil
	}
	return &wktValue
}

func temporalDomain(value *temporal) *domain.Temporal {
	if value == nil {
		return nil
	}
	return &domain.Temporal{
		Start: value.Start,
		End:   value.End,
	}
}

func (i identifierDraft) domain() domain.IdentifierInput {
	return domain.IdentifierInput{
		Type:  i.Type,
		Value: i.Value,
	}
}

func datasetResponse(value domain.DatasetVersion, includeContacts bool) record {
	return record{
		ResourceID: value.DatasetID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  string(value.CreatedBy),
		GroupID:    value.Group.ID,
		GroupName:  value.Group.Name,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
		Metadata:   datasetMetadataDTO(value.Metadata, includeContacts),
	}
}

func datasetMetadataDTO(value domain.DatasetMetadata, includeContacts bool) metadataResponse {
	result := metadataResponse{
		ID:                    value.ID,
		Title:                 value.Title,
		Description:           value.Description,
		Keywords:              value.Keywords,
		AccrualPeriodicity:    value.AccrualPeriodicity,
		QualityAnnotation:     value.QualityAnnotation,
		AccessRight:           value.AccessRight,
		Distributions:         make([]distribution, 0, len(value.Distributions)),
		SpatialCoverage:       spatialResponse(value.SpatialCoverage),
		TemporalCoverage:      temporalResponse(value.TemporalCoverage),
		PersistentIdentifiers: make([]identifier, 0, len(value.PersistentIdentifiers)),
		ContactPoints:         nil,
		Providers:             make([]resourceReference, 0, len(value.Providers)),
		Categories:            make([]resourceReference, 0, len(value.Categories)),
		Created:               value.SourceCreated,
		Updated:               value.SourceUpdated,
	}
	for _, item := range value.Distributions {
		result.Distributions = append(result.Distributions, distributionDTO(item, includeContacts))
	}
	for _, item := range value.PersistentIdentifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, identifierResponse(item))
	}
	if includeContacts {
		contacts := make([]uuid.UUID, 0, len(value.ContactPoints))
		for _, item := range value.ContactPoints {
			contacts = append(contacts, item.ID)
		}
		result.ContactPoints = &contacts
	}
	for _, item := range value.Providers {
		result.Providers = append(result.Providers, resourceReferenceDTO(item.ID, item.Name))
	}
	for _, item := range value.Categories {
		result.Categories = append(result.Categories, resourceReferenceDTO(item.ID, item.Name))
	}
	return result
}

func distributionDTO(value domain.Distribution, includeContacts bool) distribution {
	result := distribution{
		ID:               value.ID,
		Title:            value.Title,
		Description:      value.Description,
		License:          value.License,
		Type:             value.Type,
		Issued:           value.Issued,
		Modified:         value.Modified,
		ByteSize:         value.ByteSize,
		WebService:       nil,
		DownloadableFile: nil,
	}
	if value.WebService != nil {
		item := webServiceDTO(*value.WebService, includeContacts)
		result.WebService = &item
	}
	if value.DownloadableFile != nil {
		result.DownloadableFile = &downloadableFile{
			URL:    value.DownloadableFile.URL,
			Format: value.DownloadableFile.Format,
		}
	}
	return result
}

func webServiceDTO(value domain.WebService, includeContacts bool) webService {
	result := webService{
		ID:               value.ID,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateURL,
		Parameters:       make([]parameterResponse, 0, len(value.Parameters)),
		Format:           value.Format,
		Provider:         nil,
		Documentation:    nil,
		SpatialCoverage:  spatialResponse(value.SpatialCoverage),
		TemporalCoverage: temporalResponse(value.TemporalCoverage),
		ContactPoints:    nil,
	}
	for _, item := range value.Parameters {
		result.Parameters = append(result.Parameters, parameterDTO(item))
	}
	if value.Provider != nil {
		item := resourceReferenceDTO(value.Provider.ID, value.Provider.Name)
		result.Provider = &item
	}
	if value.Documentation != nil {
		result.Documentation = &documentation{
			Title:       value.Documentation.Title,
			Description: value.Documentation.Description,
			URI:         value.Documentation.URI,
		}
	}
	if includeContacts {
		contacts := make([]uuid.UUID, 0, len(value.ContactPoints))
		for _, item := range value.ContactPoints {
			contacts = append(contacts, item.ID)
		}
		result.ContactPoints = &contacts
	}
	return result
}

func parameterDTO(value domain.Parameter) parameterResponse {
	metadataValue := parameterMetadataResponse(value.GetMetadata())
	switch item := value.(type) {
	case domain.StringParameter:
		return stringParameter{
			Kind:          item.Kind(),
			Metadata:      metadataValue,
			Default:       item.Default,
			AllowedValues: nonNilSlice(item.AllowedValues),
			Pattern:       item.Pattern,
		}
	case domain.BooleanParameter:
		return booleanParameter{
			Kind:     item.Kind(),
			Metadata: metadataValue,
			Default:  item.Default,
		}
	case domain.IntegerParameter:
		return integerParameter{
			Kind:          item.Kind(),
			Metadata:      metadataValue,
			Minimum:       item.Minimum,
			Maximum:       item.Maximum,
			Default:       item.Default,
			AllowedValues: nonNilSlice(item.AllowedValues),
		}
	case domain.FloatParameter:
		return floatParameter{
			Kind:          item.Kind(),
			Metadata:      metadataValue,
			Minimum:       item.Minimum,
			Maximum:       item.Maximum,
			Default:       item.Default,
			AllowedValues: nonNilSlice(item.AllowedValues),
		}
	case domain.DateTimeParameter:
		return dateTimeParameter{
			Kind:     item.Kind(),
			Metadata: metadataValue,
			Minimum:  item.Minimum,
			Maximum:  item.Maximum,
			Default:  item.Default,
			Pattern:  item.Pattern,
		}
	default:
		return nil
	}
}

func parameterMetadataResponse(value domain.ParameterMetadata) parameterMetadata {
	return parameterMetadata{
		ID:             value.ID,
		Kind:           value.Kind,
		Variable:       value.Variable,
		Property:       value.Property,
		Label:          value.Label,
		Required:       value.Required,
		MultipleValues: value.MultipleValues,
		ReadOnly:       value.ReadOnly,
	}
}

func resourceReferenceDTO(id uuid.UUID, name string) resourceReference {
	return resourceReference{ID: id, Name: name}
}

func identifierResponse(value domain.Identifier) identifier {
	return identifier{
		ID:    value.ID,
		Type:  value.Type,
		Value: value.Value,
	}
}

func temporalResponse(value *domain.Temporal) *temporal {
	if value == nil {
		return nil
	}
	return &temporal{
		Start: value.Start,
		End:   value.End,
	}
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
