package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
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
	reader := metadatareader.New(detail.GetProperties())

	label, err := reader.RequiredString("skos:prefLabel")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme label: %w", err)
	}

	name, err := reader.RequiredString("dct:title")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme name: %w", err)
	}

	topConcepts, err := reader.ResourceRefs("skos:hasTopConcept", api.ConceptType)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme top concepts: %w", err)
	}

	orderNumber, err := metadatareader.ParseOptionalInt32(reader.OptionalString("schema:orderItemNumber"))
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme order number: %w", err)
	}

	return api.ConceptScheme{
		ID:          detail.GetId(),
		Label:       label,
		Name:        name,
		Description: reader.OptionalString("dct:description"),
		Logo:        reader.OptionalString("foaf:logo"),
		Homepage:    reader.OptionalString("foaf:homepage"),
		Color:       reader.OptionalString("schema:color"),
		OrderNumber: orderNumber,
		TopConcepts: topConcepts,
		Children:    nil,
	}, nil
}
