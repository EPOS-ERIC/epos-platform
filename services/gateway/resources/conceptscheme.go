package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
)

func (s *Service) GetConceptSchemeByID(ctx context.Context, id string) (api.ConceptScheme, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.ConceptSchemeType), id)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("fetch concept scheme '%s' detail: %w", id, err)
	}

	conceptScheme, err := mapConceptScheme(detail)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("map concept scheme '%s' detail: %w", id, err)
	}

	return conceptScheme, nil
}

func mapConceptScheme(detail *metadataclient.AccessDetailResponse) (api.ConceptScheme, error) {
	reader := newPropertyReader(detail.GetProperties())

	label, err := reader.requiredStringOneOf("prefLabel", "label")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme label: %w", err)
	}

	name, err := reader.requiredStringOneOf("title", "name")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme name: %w", err)
	}

	topConcepts, err := reader.resourceRefs("hasTopConcept", api.ConceptType)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme top concepts: %w", err)
	}

	orderNumber, err := parseOptionalInt32(reader.optionalString("orderItemNumber"))
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme order number: %w", err)
	}

	return api.ConceptScheme{
		ID:          detail.GetId(),
		Label:       label,
		Name:        name,
		Description: reader.optionalString("description"),
		Logo:        reader.optionalString("logo"),
		Homepage:    reader.optionalString("homepage"),
		Color:       reader.optionalString("color"),
		OrderNumber: orderNumber,
		TopConcepts: topConcepts,
	}, nil
}

func parseOptionalInt32(value string) (int32, error) {
	if value == "" {
		return 0, nil
	}

	parsedValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse int32 '%s': %w", value, err)
	}

	return int32(parsedValue), nil
}
