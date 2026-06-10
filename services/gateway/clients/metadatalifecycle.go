package clients

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func NewMetadatalifecycle(baseURL string, upstreamCfg config.Upstream) (*metadataclient.APIClient, error) {
	normalizedBaseURL, err := normalizeURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata-lifecycle base URL: %w", err)
	}

	httpClient, err := newUpstreamHTTPClient(upstreamCfg)
	if err != nil {
		return nil, err
	}

	cfg := metadataclient.NewConfiguration()
	cfg.Servers = metadataclient.ServerConfigurations{{
		URL:         normalizedBaseURL,
		Description: "metadata-lifecycle",
		Variables:   nil,
	}}
	cfg.HTTPClient = httpClient

	return metadataclient.NewAPIClient(cfg), nil
}
