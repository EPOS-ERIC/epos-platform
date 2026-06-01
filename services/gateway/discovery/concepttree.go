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

func (s *Service) GetConceptTree(ctx context.Context) (api.ConceptTreeResponse, error) {
	builder := conceptTreeBuilder{
		service:        s,
		conceptDetails: make(map[string]*metadataclient.AccessDetailResponse),
		conceptTrees:   make(map[string]api.ConceptTreeNode),
	}

	schemeIDs, err := builder.listConceptSchemeIDs(ctx)
	if err != nil {
		return api.ConceptTreeResponse{}, err
	}

	s.log.Debug("building concept tree", "scheme_count", len(schemeIDs))

	schemes := make([]api.ConceptScheme, len(schemeIDs))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.fanOutLimit)

	for i, schemeID := range schemeIDs {
		g.Go(func() error {
			scheme, err := builder.buildConceptScheme(groupCtx, schemeID)
			if err != nil {
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

	sort.SliceStable(schemes, func(i, j int) bool {
		leftHasOrder := schemes[i].OrderNumber > 0
		rightHasOrder := schemes[j].OrderNumber > 0
		if leftHasOrder != rightHasOrder {
			return leftHasOrder
		}
		if !leftHasOrder {
			return false
		}
		return schemes[i].OrderNumber < schemes[j].OrderNumber
	})

	return api.ConceptTreeResponse{Schemes: schemes}, nil
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

	children, err := b.buildTopConceptTrees(ctx, topConcepts)
	if err != nil {
		return api.ConceptScheme{}, err
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
		Children:    children,
	}, nil
}

func (b *conceptTreeBuilder) buildTopConceptTrees(ctx context.Context, topConcepts []api.ResourceRef) ([]api.ConceptTreeNode, error) {
	if len(topConcepts) == 0 {
		return nil, nil
	}

	children := make([]api.ConceptTreeNode, len(topConcepts))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(b.service.fanOutLimit)

	for i, topConcept := range topConcepts {
		g.Go(func() error {
			child, err := b.buildConceptTree(groupCtx, topConcept.ID, nil)
			if err != nil {
				return fmt.Errorf("build top concept '%s': %w", topConcept.ID, err)
			}

			children[i] = child
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("build top concept trees: %w", err)
	}

	return children, nil
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
		return api.ConceptTreeNode{}, err
	}

	reader := metadatareader.New(detail.GetProperties())
	name, err := reader.RequiredStringOneOf("prefLabel", "name")
	if err != nil {
		return api.ConceptTreeNode{}, fmt.Errorf("get concept name: %w", err)
	}

	childRefs, err := reader.ResourceRefs("narrower", api.ConceptType)
	if err != nil {
		return api.ConceptTreeNode{}, fmt.Errorf("get concept children: %w", err)
	}

	children, err := b.buildChildConceptTrees(ctx, childRefs, append(ancestors, id))
	if err != nil {
		return api.ConceptTreeNode{}, err
	}

	concept := api.ConceptTreeNode{
		ID:          detail.GetId(),
		Name:        name,
		Description: reader.OptionalString("definition"),
		Children:    children,
	}

	b.mu.Lock()
	b.conceptTrees[id] = concept
	b.mu.Unlock()

	return concept, nil
}

func (b *conceptTreeBuilder) buildChildConceptTrees(ctx context.Context, childRefs []api.ResourceRef, ancestors []string) ([]api.ConceptTreeNode, error) {
	if len(childRefs) == 0 {
		return nil, nil
	}

	children := make([]api.ConceptTreeNode, len(childRefs))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(b.service.fanOutLimit)

	for i, childRef := range childRefs {
		g.Go(func() error {
			child, err := b.buildConceptTree(groupCtx, childRef.ID, ancestors)
			if err != nil {
				return fmt.Errorf("build child concept '%s': %w", childRef.ID, err)
			}

			children[i] = child
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("build child concept trees: %w", err)
	}

	return children, nil
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
