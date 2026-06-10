package clients

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/usergroupclient"
)

func NewUsergroup(baseURL string, upstreamCfg config.Upstream) (*usergroupclient.APIClient, error) {
	normalizedBaseURL, err := normalizeURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize usergroup base URL: %w", err)
	}

	httpClient, err := newUpstreamHTTPClient(upstreamCfg)
	if err != nil {
		return nil, err
	}

	cfg := usergroupclient.NewConfiguration()
	cfg.Servers = usergroupclient.ServerConfigurations{{
		URL:         normalizedBaseURL,
		Description: "usergroup",
		Variables:   nil,
	}}
	cfg.HTTPClient = httpClient

	return usergroupclient.NewAPIClient(cfg), nil
}
