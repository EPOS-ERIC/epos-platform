package discovery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
	"golang.org/x/sync/errgroup"
)

const conceptSchemeSearchPageSize int32 = 100

var errConceptTreeParse = errors.New("concept tree parse")

func (s *Service) GetConceptTree(ctx context.Context) (api.ConceptTreeResponse, error) {
	builder := conceptTreeBuilder{
		service:        s,
		mu:             sync.Mutex{},
		conceptDetails: make(map[string]*metadataclient.AccessDetailResponse),
		conceptTrees:   make(map[string]api.ConceptTreeNode),
	}

	schemeIDs, err := builder.listConceptSchemeIDs(ctx)
	if err != nil {
		return api.ConceptTreeResponse{}, fmt.Errorf("list concept scheme ids: %w", err)
	}

	s.log.Debug("building concept tree", "scheme_count", len(schemeIDs))

	schemes := make([]api.ConceptScheme, len(schemeIDs))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.fanOutLimit)

	for i, schemeID := range schemeIDs {
		g.Go(func() error {
			scheme, err := builder.buildConceptScheme(groupCtx, schemeID)
			if err != nil {
				if errors.Is(err, errConceptTreeParse) {
					s.log.WarnContext(groupCtx, "skip malformed concept scheme", "scheme_id", schemeID, "error", err)
					return nil
				}
				return fmt.Errorf("build concept scheme '%s': %w", schemeID, err)
			}

			schemes[i] = scheme
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return api.ConceptTreeResponse{}, fmt.Errorf("build concept tree timeout: %w: %w", ErrUpstreamTimeout, err)
		}

		return api.ConceptTreeResponse{}, fmt.Errorf("build concept tree request failed: %w: %w", ErrUpstreamRequestFailed, err)
	}

	filteredSchemes := make([]api.ConceptScheme, 0, len(schemes))
	for _, scheme := range schemes {
		if scheme.ID == "" {
			continue
		}
		filteredSchemes = append(filteredSchemes, scheme)
	}

	sort.SliceStable(filteredSchemes, func(i, j int) bool {
		leftHasOrder := filteredSchemes[i].OrderNumber > 0
		rightHasOrder := filteredSchemes[j].OrderNumber > 0
		if leftHasOrder != rightHasOrder {
			return leftHasOrder
		}
		if !leftHasOrder {
			return false
		}
		return filteredSchemes[i].OrderNumber < filteredSchemes[j].OrderNumber
	})

	return api.ConceptTreeResponse{Schemes: filteredSchemes}, nil
}

type conceptTreeBuilder struct {
	service *Service

	mu             sync.Mutex
	conceptDetails map[string]*metadataclient.AccessDetailResponse
	conceptTrees   map[string]api.ConceptTreeNode
}

func (b *conceptTreeBuilder) listConceptSchemeIDs(ctx context.Context) ([]string, error) {
	var schemeIDs []string

	for page := int32(1); ; page++ {
		resp, _, err := b.service.metadataClient.AccessAPI.
			ApiV2AccessSearchShapeGet(ctx, string(api.ConceptSchemeType)).
			Page(page).
			ResultsPerPage(conceptSchemeSearchPageSize).
			Execute()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, fmt.Errorf("list concept schemes timeout: %w: %w", ErrUpstreamTimeout, err)
			}

			return nil, fmt.Errorf("list concept schemes request failed: %w: %w", ErrUpstreamRequestFailed, err)
		}

		if resp == nil {
			return nil, fmt.Errorf("list concept schemes: empty response")
		}

		for _, item := range resp.Items {
			if item.Id == "" {
				continue
			}
			if slices.Contains(schemeIDs, item.Id) {
				continue
			}
			schemeIDs = append(schemeIDs, item.Id)
		}

		if page >= resp.TotalPages || len(resp.Items) == 0 {
			break
		}
	}

	return schemeIDs, nil
}

func (b *conceptTreeBuilder) buildConceptScheme(ctx context.Context, id string) (api.ConceptScheme, error) {
	detail, err := metadataaccess.FetchDetail(ctx, b.service.metadataClient, string(api.ConceptSchemeType), id)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("fetch concept scheme '%s' details: %w", id, err)
	}

	reader := metadatareader.New(detail.GetProperties())

	label, err := reader.RequiredString("skos:prefLabel")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("%w: get concept scheme label: %w", errConceptTreeParse, err)
	}

	name, err := reader.RequiredString("dct:title")
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("%w: get concept scheme name: %w", errConceptTreeParse, err)
	}

	topConcepts, err := reader.ResourceRefs("skos:hasTopConcept", api.ConceptType)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("%w: get concept scheme top concepts: %w", errConceptTreeParse, err)
	}

	orderNumber, err := metadatareader.ParseOptionalInt32(reader.OptionalString("schema:orderItemNumber"))
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("%w: get concept scheme order number: %w", errConceptTreeParse, err)
	}

	children, err := b.buildTopConceptTrees(ctx, topConcepts)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("build concept scheme '%s' top concept trees: %w", id, err)
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
		Children:    children,
	}, nil
}

func (b *conceptTreeBuilder) buildTopConceptTrees(ctx context.Context, topConcepts []api.ResourceRef) ([]api.ConceptTreeNode, error) {
	return b.buildConceptTrees(ctx, topConcepts, nil, "top", "build top concept trees")
}

func (b *conceptTreeBuilder) buildConceptTree(ctx context.Context, id string, ancestors []string) (api.ConceptTreeNode, error) {
	if slices.Contains(ancestors, id) {
		return api.ConceptTreeNode{}, fmt.Errorf("cycle detected in concept tree at '%s'", id)
	}

	b.mu.Lock()
	if concept, ok := b.conceptTrees[id]; ok {
		b.mu.Unlock()
		return concept, nil
	}
	b.mu.Unlock()

	detail, err := b.fetchConceptDetail(ctx, id)
	if err != nil {
		return api.ConceptTreeNode{}, fmt.Errorf("fetch concept tree node '%s': %w", id, err)
	}

	reader := metadatareader.New(detail.GetProperties())
	name, err := reader.RequiredString("skos:prefLabel")
	if err != nil {
		return api.ConceptTreeNode{}, fmt.Errorf("%w: get concept name: %w", errConceptTreeParse, err)
	}

	childRefs, err := reader.ResourceRefs("skos:narrower", api.ConceptType)
	if err != nil {
		return api.ConceptTreeNode{}, fmt.Errorf("%w: get concept children: %w", errConceptTreeParse, err)
	}

	children, err := b.buildChildConceptTrees(ctx, childRefs, append(ancestors, id))
	if err != nil {
		return api.ConceptTreeNode{}, fmt.Errorf("build concept '%s' child trees: %w", id, err)
	}

	concept := api.ConceptTreeNode{
		ID:          detail.GetId(),
		Name:        name,
		Description: reader.OptionalString("skos:definition"),
		Children:    children,
	}

	b.mu.Lock()
	b.conceptTrees[id] = concept
	b.mu.Unlock()

	return concept, nil
}

func (b *conceptTreeBuilder) buildChildConceptTrees(ctx context.Context, childRefs []api.ResourceRef, ancestors []string) ([]api.ConceptTreeNode, error) {
	return b.buildConceptTrees(ctx, childRefs, ancestors, "child", "build child concept trees")
}

func (b *conceptTreeBuilder) buildConceptTrees(ctx context.Context, refs []api.ResourceRef, ancestors []string, conceptType, treeErr string) ([]api.ConceptTreeNode, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	children := make([]api.ConceptTreeNode, len(refs))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(b.service.fanOutLimit)

	for i, ref := range refs {
		g.Go(func() error {
			child, err := b.buildConceptTree(groupCtx, ref.ID, ancestors)
			if err != nil {
				if errors.Is(err, errConceptTreeParse) {
					b.service.log.WarnContext(groupCtx, "skip malformed concept", "concept_type", conceptType, "concept_id", ref.ID, "error", err)
					return nil
				}
				return fmt.Errorf("build %s concept '%s': %w", conceptType, ref.ID, err)
			}

			children[i] = child
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("%s: %w", treeErr, err)
	}

	filteredChildren := make([]api.ConceptTreeNode, 0, len(children))
	for _, child := range children {
		if child.ID == "" {
			continue
		}
		filteredChildren = append(filteredChildren, child)
	}

	return filteredChildren, nil
}

func (b *conceptTreeBuilder) fetchConceptDetail(ctx context.Context, id string) (*metadataclient.AccessDetailResponse, error) {
	b.mu.Lock()
	if detail, ok := b.conceptDetails[id]; ok {
		b.mu.Unlock()
		return detail, nil
	}
	b.mu.Unlock()

	detail, err := metadataaccess.FetchDetail(ctx, b.service.metadataClient, string(api.ConceptType), id)
	if err != nil {
		return nil, fmt.Errorf("fetch concept '%s' details: %w", id, err)
	}

	b.mu.Lock()
	b.conceptDetails[id] = detail
	b.mu.Unlock()

	return detail, nil
}
