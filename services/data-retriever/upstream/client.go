package upstream

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retrieval"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(headerTimeout time.Duration) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = headerTimeout
	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// ExecuteStream performs the request and returns its unbuffered response body.
func (c *Client) ExecuteStream(ctx context.Context, req domain.PreparedRetrieval) (*retrieval.UpstreamResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.GeneratedURL, bytes.NewReader(req.BodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", req.Body.ContentType)
	}
	if strings.TrimSpace(req.BearerToken) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.BearerToken)
	}

	response, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute upstream request: %w", err)
	}

	contentType := response.Header.Get("Content-Type")
	disposition := strings.TrimSpace(response.Header.Get("Content-Disposition"))
	if disposition == "" {
		disposition = syntheticDisposition(req.GeneratedURL)
	}
	return &retrieval.UpstreamResponse{
		Head: domain.ResponseHead{
			StatusCode:         response.StatusCode,
			ContentType:        contentType,
			ContentLength:      response.ContentLength,
			Location:           response.Header.Get("Location"),
			ContentDisposition: disposition,
			IsBinary:           isBinaryPayload(contentType),
		},
		Body: response.Body,
	}, nil
}

func syntheticDisposition(generatedURL string) string {
	name := urlFileName(generatedURL)
	if name == "" {
		return ""
	}
	return mime.FormatMediaType("inline", map[string]string{"filename": name})
}

func urlFileName(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Path == "" {
		return ""
	}
	base := path.Base(strings.TrimSuffix(parsed.Path, "/"))
	if base == "" || base == "." || base == "/" {
		return ""
	}
	name, err := url.PathUnescape(base)
	if err != nil {
		return ""
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return ""
	}
	if !strings.Contains(strings.TrimSuffix(name, "."), ".") {
		return ""
	}
	return name
}

func isBinaryPayload(contentType string) bool {
	base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(base, "text/") || base == "application/json" || base == "application/xml" || base == "application/x-www-form-urlencoded" || strings.HasSuffix(base, "+json") || strings.HasSuffix(base, "+xml") {
		return false
	}
	return true
}
