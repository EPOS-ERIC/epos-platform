package metadatalifecycle

import (
	"context"
	"net/http"
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
	Query         string
	Type          string
	Status        string
	BBox          string
	Geo           string
	GeoRelation   string
	Lat           *float64
	Lon           *float64
	Radius        *float64
	StartDate     *time.Time
	EndDate       *time.Time
	Keywords      string
	Organization  string
	Concept       string
	ConceptScheme string
	DataProvider  string
	Page          *int32
	PageSize      *int32
	Facets        *bool
	Facetstype    string
}

func NewService(baseURL string, timeout time.Duration) (*Service, error) {
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	cfg := metadataclient.NewConfiguration()
	cfg.Servers = metadataclient.ServerConfigurations{
		{
			URL:         baseURL,
			Description: "metadata-lifecycle",
		},
	}
	cfg.HTTPClient = &http.Client{Timeout: timeout}

	log.Debug("metadata lifecycle client configured", "base_url", baseURL, "timeout", timeout.String())

	return &Service{client: metadataclient.NewAPIClient(cfg)}, nil
}

func (s *Service) Search(ctx context.Context, params SearchParams) (*metadataclient.ApiV2SearchGet200Response, error) {
	req := s.client.SearchAPI.ApiV2SearchGet(ctx)

	if params.Query != "" {
		req = req.Q(params.Query)
	}
	if params.Type != "" {
		req = req.Type_(params.Type)
	}
	if params.Status != "" {
		req = req.Status(params.Status)
	}
	if params.BBox != "" {
		req = req.Bbox(params.BBox)
	}
	if params.Geo != "" {
		req = req.Geo(params.Geo)
	}
	if params.GeoRelation != "" {
		req = req.GeoRelation(params.GeoRelation)
	}
	if params.Lat != nil {
		req = req.Lat(*params.Lat)
	}
	if params.Lon != nil {
		req = req.Lon(*params.Lon)
	}
	if params.Radius != nil {
		req = req.Radius(*params.Radius)
	}
	if params.StartDate != nil {
		req = req.StartDate(*params.StartDate)
	}
	if params.EndDate != nil {
		req = req.EndDate(*params.EndDate)
	}
	if params.Keywords != "" {
		req = req.Keywords(params.Keywords)
	}
	if params.Organization != "" {
		req = req.Organization(params.Organization)
	}
	if params.Concept != "" {
		req = req.Concept(params.Concept)
	}
	if params.ConceptScheme != "" {
		req = req.ConceptScheme(params.ConceptScheme)
	}
	if params.DataProvider != "" {
		req = req.DataProvider(params.DataProvider)
	}
	if params.Page != nil {
		req = req.Page(*params.Page)
	}
	if params.PageSize != nil {
		req = req.PageSize(*params.PageSize)
	}
	if params.Facets != nil {
		req = req.Facets(*params.Facets)
	}
	if params.Facetstype != "" {
		req = req.Facetstype(params.Facetstype)
	}

	response, httpResponse, err := req.Execute()
	if err != nil {
		return nil, wrapRequestError(searchOperation, httpResponse, err)
	}

	return response, nil
}
