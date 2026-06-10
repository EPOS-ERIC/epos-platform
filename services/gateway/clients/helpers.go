// Package clients wires generated outbound API clients for gateway services.
package clients

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

func newUpstreamHTTPClient(upstreamCfg config.Upstream) (*http.Client, error) {
	upstreamClient, err := upstream.NewClient(upstream.Config{
		Timeout:     upstreamCfg.Timeout,
		RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL:    upstreamCfg.CacheTTL,
		MaxEntries:  upstreamCfg.MaxEntries,
		Transport:   nil,
		Logger:      nil,
	})
	if err != nil {
		return nil, fmt.Errorf("create upstream client: %w", err)
	}

	return upstreamClient.StandardClient(), nil
}

func normalizeURL(baseURL string) (string, error) {
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
