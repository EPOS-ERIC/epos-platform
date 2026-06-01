package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
)

func (s *Service) GetConceptByID(ctx context.Context, id string) (api.Concept, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.ConceptType), id)
	if err != nil {
		return api.Concept{}, fmt.Errorf("fetch concept '%s' detail: %w", id, err)
	}

	concept, err := mapConcept(detail)
	if err != nil {
		return api.Concept{}, fmt.Errorf("map concept '%s' detail: %w", id, err)
	}

	return concept, nil
}

func mapConcept(detail *metadataclient.AccessDetailResponse) (api.Concept, error) {
	reader := metadatareader.New(detail.GetProperties())

	name, err := reader.RequiredString("skos:prefLabel")
	if err != nil {
		return api.Concept{}, fmt.Errorf("get concept name: %w", err)
	}

	inScheme, err := reader.OptionalResourceRef("skos:inScheme", api.ConceptSchemeType)
	if err != nil {
		return api.Concept{}, fmt.Errorf("get concept in scheme: %w", err)
	}

	children, err := mapConceptChildren(reader)
	if err != nil {
		return api.Concept{}, fmt.Errorf("get concept children: %w", err)
	}

	return api.Concept{
		ID:          detail.GetId(),
		Name:        name,
		Description: reader.OptionalString("skos:definition"),
		InScheme:    inScheme,
		Children:    children,
	}, nil
}

func mapConceptChildren(reader metadatareader.Reader) ([]api.ResourceRef, error) {
	children, err := reader.ResourceRefs("skos:narrower", api.ConceptType)
	if err != nil {
		return nil, fmt.Errorf("get narrower refs: %w", err)
	}

	return children, nil
}
