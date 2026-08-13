package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
	"github.com/google/uuid"
)

type Resources struct {
	baseURL string
	http    *http.Client
}

type Dataset struct {
	ID                    uuid.UUID      `json:"id"`
	SourceIRI             string         `json:"source_iri"`
	Title                 string         `json:"title"`
	Description           string         `json:"description"`
	Keywords              string         `json:"keywords"`
	AccrualPeriodicity    *string        `json:"accrual_periodicity"`
	QualityAnnotation     *string        `json:"quality_annotation"`
	AccessRight           *string        `json:"access_right"`
	Distributions         []Distribution `json:"distributions"`
	SpatialCoverage       *Spatial       `json:"spatial_coverage"`
	TemporalCoverage      *Temporal      `json:"temporal_coverage"`
	PersistentIdentifiers []Identifier   `json:"persistent_identifiers"`
	Providers             []Organization `json:"providers"`
	Categories            []Category     `json:"categories"`
	Created               time.Time      `json:"created"`
	Updated               time.Time      `json:"updated"`
	SourceCreated         *time.Time     `json:"source_created"`
	SourceUpdated         *time.Time     `json:"source_updated"`
	Status                string         `json:"status"`
}

type Distribution struct {
	ID               uuid.UUID   `json:"id"`
	SourceIRI        string      `json:"source_iri"`
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	License          *string     `json:"license"`
	Type             string      `json:"type"`
	Issued           *time.Time  `json:"issued"`
	Modified         *time.Time  `json:"modified"`
	WebService       *WebService `json:"web_service"`
	DownloadableFile *File       `json:"downloadable_file"`
}

type File struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

type WebService struct {
	ID               uuid.UUID      `json:"id"`
	SourceIRI        string         `json:"source_iri"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	TemplateURL      string         `json:"template_url"`
	Parameters       []Parameter    `json:"parameters"`
	Format           string         `json:"format"`
	Provider         *Organization  `json:"provider"`
	Documentation    *Documentation `json:"documentation"`
	SpatialCoverage  *Spatial       `json:"spatial_coverage"`
	TemporalCoverage *Temporal      `json:"temporal_coverage"`
}

type SoftwareApplication struct {
	ID               uuid.UUID      `json:"id"`
	SourceIRI        string         `json:"source_iri"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	MainEntityOfPage *string        `json:"main_entity_of_page"`
	License          *string        `json:"license"`
	Version          *string        `json:"version"`
	Keywords         string         `json:"keywords"`
	DownloadURL      *string        `json:"download_url"`
	Requirements     *string        `json:"requirements"`
	OperatingSystem  *string        `json:"operating_system"`
	Creators         []Organization `json:"creators"`
	Identifiers      []Identifier   `json:"identifiers"`
	Categories       []Category     `json:"categories"`
}

type SoftwareSourceCode struct {
	ID                   uuid.UUID      `json:"id"`
	SourceIRI            string         `json:"source_iri"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	ProgrammingLanguage  *string        `json:"programming_language"`
	Keywords             string         `json:"keywords"`
	Repository           *string        `json:"repository"`
	License              *string        `json:"license"`
	SoftwareRequirements *string        `json:"software_requirements"`
	RuntimePlatform      *string        `json:"runtime_platform"`
	MainEntityOfPage     *string        `json:"main_entity_of_page"`
	Version              *string        `json:"version"`
	Creators             []Organization `json:"creators"`
	Identifiers          []Identifier   `json:"identifiers"`
	Categories           []Category     `json:"categories"`
}

type Parameter struct {
	Metadata      ParameterMetadata `json:"metadata"`
	Default       json.RawMessage   `json:"default"`
	Minimum       json.RawMessage   `json:"minimum"`
	Maximum       json.RawMessage   `json:"maximum"`
	AllowedValues []json.RawMessage `json:"allowed_values"`
	Pattern       string            `json:"pattern"`
}

type ParameterMetadata struct {
	SourceIRI      string `json:"source_iri"`
	Kind           string `json:"kind"`
	Variable       string `json:"variable"`
	Property       string `json:"property"`
	Label          string `json:"label"`
	Required       bool   `json:"required"`
	MultipleValues bool   `json:"multiple_values"`
	ReadOnly       bool   `json:"read_only"`
}

type Documentation struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	URI         string  `json:"uri"`
}

type Organization struct {
	ID          uuid.UUID    `json:"id"`
	SourceIRI   string       `json:"source_iri"`
	Name        string       `json:"name"`
	Street      *string      `json:"street"`
	Locality    *string      `json:"locality"`
	PostalCode  *string      `json:"postal_code"`
	Country     *string      `json:"country"`
	LogoURL     *string      `json:"logo_url"`
	PageURL     *string      `json:"page_url"`
	Email       *string      `json:"email"`
	LEICode     *string      `json:"lei_code"`
	Identifiers []Identifier `json:"identifiers"`
}

type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Category struct {
	ID        uuid.UUID      `json:"id"`
	SourceIRI string         `json:"source_iri"`
	Name      string         `json:"name"`
	Scheme    CategoryScheme `json:"scheme"`
	Children  []Category     `json:"children"`
}

type CategoryScheme struct {
	ID          uuid.UUID  `json:"id"`
	SourceIRI   string     `json:"source_iri"`
	Name        string     `json:"name"`
	Label       *string    `json:"label"`
	Logo        *string    `json:"logo"`
	Page        *string    `json:"page"`
	Color       *string    `json:"color"`
	Position    int32      `json:"position"`
	TopConcepts []Category `json:"top_concepts"`
}

type Spatial struct {
	Point   *Coordinate  `json:"point"`
	Polygon []Coordinate `json:"polygon"`
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Temporal struct {
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
}

func NewResources(baseURL string, upstreamCfg config.Upstream) (*Resources, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize resources base URL: %w", err)
	}

	upstreamClient, err := upstream.NewClient(upstream.Config{
		Service: "backoffice", Timeout: upstreamCfg.Timeout, RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL: upstreamCfg.CacheTTL, MaxEntries: upstreamCfg.MaxEntries, Transport: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("create upstream client: %w", err)
	}

	return &Resources{baseURL: normalizedBaseURL, http: upstreamClient.StandardClient()}, nil
}

func (c *Resources) GetDataset(ctx context.Context, id uuid.UUID) (Dataset, error) {
	var resource Dataset
	body, err := c.get(ctx, "dataset", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources dataset '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) GetDistribution(ctx context.Context, id uuid.UUID) (Distribution, error) {
	var resource Distribution
	body, err := c.get(ctx, "distribution", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources distribution '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) GetWebService(ctx context.Context, id uuid.UUID) (WebService, error) {
	var resource WebService
	body, err := c.get(ctx, "webservice", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources webservice '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) GetSoftwareApplication(ctx context.Context, id uuid.UUID) (SoftwareApplication, error) {
	var resource SoftwareApplication
	body, err := c.get(ctx, "softwareapplication", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources software application '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) GetSoftwareSourceCode(ctx context.Context, id uuid.UUID) (SoftwareSourceCode, error) {
	var resource SoftwareSourceCode
	body, err := c.get(ctx, "softwaresourcecode", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources software source code '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) GetOrganization(ctx context.Context, id uuid.UUID) (Organization, error) {
	var resource Organization
	body, err := c.get(ctx, "organization", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources organization '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) GetConcept(ctx context.Context, id uuid.UUID) (Category, error) {
	var resource Category
	body, err := c.get(ctx, "concept", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources concept '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) GetConceptScheme(ctx context.Context, id uuid.UUID) (CategoryScheme, error) {
	var resource CategoryScheme
	body, err := c.get(ctx, "conceptscheme", id)
	if err != nil {
		return resource, err
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return resource, fmt.Errorf("decode resources conceptscheme '%s': %w", id, err)
	}
	return resource, nil
}

func (c *Resources) get(ctx context.Context, resourceType string, id uuid.UUID) ([]byte, error) {
	if c == nil || c.http == nil {
		return nil, fmt.Errorf("resources client is not configured")
	}

	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse resources base URL: %w", err)
	}

	endpoint.Path, err = url.JoinPath(endpoint.Path, "/api/v2/resources/", resourceType, id.String())
	if err != nil {
		return nil, fmt.Errorf("join resources path URL: %w", err)
	}
	endpoint.RawPath = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create resources request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("get resources %s '%s': %w: %w", resourceType, id, domain.ErrUpstreamTimeout, err)
		}
		return nil, fmt.Errorf("get resources %s '%s': %w", resourceType, id, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK: // Continue reading response body.
	case http.StatusBadRequest:
		return nil, domain.ErrBadRequest
	case http.StatusNotFound:
		return nil, domain.ErrNotFound
	default:
		return nil, fmt.Errorf("get resources %s '%s': unexpected status %s", resourceType, id, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read resources %s '%s': %w", resourceType, id, err)
	}
	return body, nil
}
