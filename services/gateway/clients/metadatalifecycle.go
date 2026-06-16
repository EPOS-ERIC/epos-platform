package clients

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

func NewMetadatalifecycle(baseURL string, upstreamCfg config.Upstream) (*metadataclient.APIClient, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata-lifecycle base URL: %w", err)
	}

	upstreamClient, err := upstream.NewClient(upstream.Config{
		Timeout:     upstreamCfg.Timeout,
		RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL:    upstreamCfg.CacheTTL,
		MaxEntries:  upstreamCfg.MaxEntries,
		Transport:   nil,
		Logger:      logging.Get("metadata-lifecycle-client"),
	})
	if err != nil {
		return nil, fmt.Errorf("create upstream client: %w", err)
	}
	httpClient := upstreamClient.StandardClient()

	cfg := metadataclient.NewConfiguration()
	cfg.Servers = metadataclient.ServerConfigurations{{
		URL:         normalizedBaseURL,
		Description: "metadata-lifecycle",
		Variables:   nil,
	}}
	cfg.HTTPClient = httpClient

	return metadataclient.NewAPIClient(cfg), nil
}
