package clients

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

const defaultMetadataRequestTimeout = 30 * time.Second

func NewMetadatalifecycle(baseURL string, timeout time.Duration) (*metadataclient.APIClient, error) {
	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata-lifecycle base URL: %w", err)
	}

	if timeout <= 0 {
		timeout = defaultMetadataRequestTimeout
	}

	cfg := metadataclient.NewConfiguration()
	cfg.Servers = metadataclient.ServerConfigurations{{
		URL:         normalizedBaseURL,
		Description: "metadata-lifecycle",
	}}
	cfg.HTTPClient = &http.Client{Timeout: timeout}

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
