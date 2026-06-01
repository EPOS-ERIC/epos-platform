package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
)

func loadAccessDetailFixture(t *testing.T, relativePath string) *metadataclient.AccessDetailResponse {
	t.Helper()
	return testfixtures.LoadAccessDetail(t, relativePath)
}

func newFixtureAccessDetailFetcher(t *testing.T, fixtures map[string]string) accessDetailFetcher {
	t.Helper()

	loaded := make(map[string]*metadataclient.AccessDetailResponse, len(fixtures))
	for key, fixturePath := range fixtures {
		loaded[key] = loadAccessDetailFixture(t, fixturePath)
	}

	return func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx

		key := fixtureKey(shape, id)
		detail, ok := loaded[key]
		if !ok {
			return nil, fmt.Errorf("missing fixture for %s", key)
		}

		return detail, nil
	}
}

func fixtureKey(shape string, id string) string {
	return shape + "::" + id
}
