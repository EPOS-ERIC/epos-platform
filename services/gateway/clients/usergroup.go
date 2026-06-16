package clients

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/usergroupclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

func NewUsergroup(baseURL string, upstreamCfg config.Upstream) (*usergroupclient.APIClient, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize usergroup base URL: %w", err)
	}

	upstreamClient, err := upstream.NewClient(upstream.Config{
		Timeout:     upstreamCfg.Timeout,
		RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL:    0, // Disable caching for the user group client
		MaxEntries:  upstreamCfg.MaxEntries,
		Transport:   nil,
		Logger:      logging.Get("usergroup-client"),
	})
	if err != nil {
		return nil, fmt.Errorf("create upstream client: %w", err)
	}
	httpClient := upstreamClient.StandardClient()

	cfg := usergroupclient.NewConfiguration()
	cfg.Servers = usergroupclient.ServerConfigurations{{
		URL:         normalizedBaseURL,
		Description: "usergroup",
		Variables:   nil,
	}}
	cfg.HTTPClient = httpClient

	return usergroupclient.NewAPIClient(cfg), nil
}
