package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
	"golang.org/x/sync/errgroup"
)

func (s *Service) GetOrganizationByID(ctx context.Context, id string) (api.Organization, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.OrganizationType), id)
	if err != nil {
		return api.Organization{}, fmt.Errorf("fetch organization '%s' detail: %w", id, err)
	}

	organization, err := mapOrganization(ctx, detail, fetchDetail)
	if err != nil {
		return api.Organization{}, fmt.Errorf("map organization '%s' detail: %w", id, err)
	}

	return organization, nil
}

func mapOrganization(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher) (api.Organization, error) {
	properties := detail.GetProperties()
	reader := metadatareader.New(properties)

	name, err := reader.RequiredString("schema:legalName")
	if err != nil {
		return api.Organization{}, fmt.Errorf("get organization name: %w", err)
	}

	identifiers, address, err := aggregateOrganizationEntities(ctx, properties, fetch)
	if err != nil {
		return api.Organization{}, err
	}

	contactPoints, err := reader.PassthroughRefs("schema:contactPoint")
	if err != nil {
		return api.Organization{}, fmt.Errorf("get organization contact points: %w", err)
	}

	memberOf, err := reader.ResourceRefs("schema:memberOf", api.OrganizationType)
	if err != nil {
		return api.Organization{}, fmt.Errorf("get organization member of: %w", err)
	}

	email, err := reader.Strings("schema:email")
	if err != nil {
		return api.Organization{}, fmt.Errorf("get organization email: %w", err)
	}

	telephone, err := reader.Strings("schema:telephone")
	if err != nil {
		return api.Organization{}, fmt.Errorf("get organization telephone: %w", err)
	}

	return api.Organization{
		ID:            detail.GetId(),
		Identifiers:   identifiers,
		Name:          name,
		LeiCode:       reader.OptionalString("epos:leiCode"),
		Address:       address,
		Logo:          reader.OptionalString("schema:logo"),
		URL:           reader.OptionalString("schema:url"),
		Email:         email,
		Telephone:     telephone,
		ContactPoints: contactPoints,
		MemberOf:      memberOf,
	}, nil
}

func aggregateOrganizationEntities(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) ([]api.Identifier, string, error) {
	var (
		identifiers []api.Identifier
		address     string
	)

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		mappedIdentifiers, err := aggregateIdentifiers(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get organization identifiers: %w", err)
		}
		identifiers = mappedIdentifiers
		return nil
	})
	g.Go(func() error {
		mappedAddress, err := aggregateOrganizationAddress(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get organization address: %w", err)
		}
		address = mappedAddress
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, "", fmt.Errorf("aggregate organization entities: %w", err)
	}

	return identifiers, address, nil
}

func aggregateOrganizationAddress(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) (string, error) {
	reader := metadatareader.New(properties)
	items, err := reader.Items("schema:address")
	if err != nil {
		return "", fmt.Errorf("get address items: %w", err)
	}
	if len(items) == 0 {
		return "", nil
	}

	item := items[0]
	if item.Value != "" {
		return item.Value, nil
	}
	if item.ID == "" {
		return "", nil
	}

	detail, err := fetch(ctx, metadatareader.FirstNonEmpty(item.Type, "postaladdress"), item.ID)
	if err != nil {
		return "", err
	}

	addressReader := metadatareader.New(detail.GetProperties())
	parts := []string{
		addressReader.OptionalString("schema:streetAddress"),
		addressReader.OptionalString("schema:addressLocality"),
		addressReader.OptionalString("schema:postalCode"),
		addressReader.OptionalString("schema:addressCountry"),
	}

	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		nonEmpty = append(nonEmpty, part)
	}

	return strings.Join(nonEmpty, ", "), nil
}
