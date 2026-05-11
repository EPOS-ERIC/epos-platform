package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

// TODO: define the logging component naming strategy to use in the repo
var log = logging.Get("metadata lifecycle service")

const (
	defaultRequestTimeout = 30 * time.Second
	searchOperation       = "search metadata"
)

type Service struct {
	client *metadataclient.APIClient
}

type SearchParams struct {
	Query            string
	Page             *int32
	ResultsPerPage   *int32
	Type             []metadataclient.ResourceType
	SpatialFilter    string
	DateFrom         *time.Time
	DateTo           *time.Time
	ServiceProvider  []string
	VersioningStatus []metadataclient.VersioningStatus
	Keywords         []string
}

func NewService(baseURL string, timeout time.Duration) (*Service, error) {
	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata-lifecycle base URL: %w", err)
	}

	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	cfg := metadataclient.NewConfiguration()
	cfg.Servers = metadataclient.ServerConfigurations{
		{
			URL:         normalizedBaseURL,
			Description: "metadata-lifecycle",
		},
	}
	cfg.HTTPClient = &http.Client{Timeout: timeout}

	log.Debug("metadata lifecycle client configured", "base_url", normalizedBaseURL, "timeout", timeout.String())

	return &Service{client: metadataclient.NewAPIClient(cfg)}, nil
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

func (s *Service) Search(ctx context.Context, params SearchParams) (*metadataclient.SearchResponse, error) {
	req := s.client.SearchAPI.SearchMetadata(ctx)

	if params.Query != "" {
		req = req.Q(params.Query)
	}
	if params.Page != nil {
		req = req.Page(*params.Page)
	}
	if params.ResultsPerPage != nil {
		req = req.ResultsPerPage(*params.ResultsPerPage)
	}
	if len(params.Type) > 0 {
		req = req.Type_(params.Type)
	}
	if params.SpatialFilter != "" {
		req = req.SpatialFilter(params.SpatialFilter)
	}
	if params.DateFrom != nil {
		req = req.DateFrom(*params.DateFrom)
	}
	if params.DateTo != nil {
		req = req.DateTo(*params.DateTo)
	}
	if len(params.ServiceProvider) > 0 {
		req = req.ServiceProvider(params.ServiceProvider)
	}
	if len(params.VersioningStatus) > 0 {
		req = req.VersioningStatus(params.VersioningStatus)
	}
	if len(params.Keywords) > 0 {
		req = req.Keywords(params.Keywords)
	}

	response, httpResponse, err := req.Execute()
	if err != nil {
		return nil, wrapRequestError(searchOperation, httpResponse, err)
	}

	log.Debug("search response received", "response", response)

	return response, nil
}
