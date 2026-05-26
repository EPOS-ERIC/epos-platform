package clients

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

const defaultMetadataRequestTimeout = 30 * time.Second

func NewMetadatalifecycle(baseURL string, timeout time.Duration, upstreamCfg config.Upstream) (*metadataclient.APIClient, error) {
	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata-lifecycle base URL: %w", err)
	}

	if timeout <= 0 {
		timeout = defaultMetadataRequestTimeout
	}

	clientTimeout := timeout
	if upstreamCfg.Timeout > 0 {
		clientTimeout = upstreamCfg.Timeout
	}

	upstreamClient, err := upstream.NewClient(upstream.Config{
		Timeout:     clientTimeout,
		RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL:    upstreamCfg.CacheTTL,
		MaxEntries:  upstreamCfg.MaxEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("create upstream client: %w", err)
	}

	cfg := metadataclient.NewConfiguration()
	cfg.Servers = metadataclient.ServerConfigurations{{
		URL:         normalizedBaseURL,
		Description: "metadata-lifecycle",
	}}
	cfg.HTTPClient = upstreamClient.StandardClient()

	return metadataclient.NewAPIClient(cfg), nil
}

func normalizeBaseURL(baseURL string) (string, error) {
	trimmedBaseURL := strings.TrimSpace(baseURL)
	if trimmedBaseURL == "" {
		return "", fmt.Errorf("base URL is required")
	}

	parsedURL, err := url.Parse(trimmedBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}

	if !parsedURL.IsAbs() || parsedURL.Host == "" {
		return "", fmt.Errorf("base URL must be an absolute URL")
	}

	return strings.TrimSuffix(parsedURL.String(), "/"), nil
}
