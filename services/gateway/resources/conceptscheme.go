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

	label, err := reader.RequiredStringOneOf("prefLabel", "label")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme label: %w", err)
	}

	name, err := reader.RequiredStringOneOf("title", "name")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme name: %w", err)
	}

	topConcepts, err := reader.ResourceRefs("hasTopConcept", api.ConceptType)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme top concepts: %w", err)
	}

	orderNumber, err := metadatareader.ParseOptionalInt32(reader.OptionalString("orderItemNumber"))
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("get concept scheme order number: %w", err)
	}

	return api.ConceptScheme{
		ID:          detail.GetId(),
		Label:       label,
		Name:        name,
		Description: reader.OptionalString("description"),
		Logo:        reader.OptionalString("logo"),
		Homepage:    reader.OptionalString("homepage"),
		Color:       reader.OptionalString("color"),
		OrderNumber: orderNumber,
		TopConcepts: topConcepts,
	}, nil
}
