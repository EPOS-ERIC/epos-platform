package upstream

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retrieval"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(headerTimeout time.Duration) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.ResponseHeaderTimeout = headerTimeout
	return &Client{
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(transport),
		},
	}
}

// ExecuteStream performs the request and returns its unbuffered response body.
func (c *Client) ExecuteStream(ctx context.Context, req domain.PreparedRetrieval) (*retrieval.UpstreamResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	response, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute upstream request: %w", err)
	}

	contentType := response.Header.Get("Content-Type")
	return &retrieval.UpstreamResponse{
		Head: domain.ResponseHead{
			StatusCode:    response.StatusCode,
			ContentType:   contentType,
			ContentLength: response.ContentLength,
			IsBinary:      isBinaryPayload(contentType),
		},
		Body: response.Body,
	}, nil
}

func isBinaryPayload(contentType string) bool {
	base, _, _ := strings.Cut(contentType, ";")
	base = strings.ToLower(strings.TrimSpace(base))
	if strings.HasPrefix(base, "text/") {
		return false
	}
	if base == "application/json" || base == "application/xml" || base == "application/x-www-form-urlencoded" {
		return false
	}
	if strings.HasSuffix(base, "+json") || strings.HasSuffix(base, "+xml") {
		return false
	}
	return true
}
