package retriever

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
)

// StreamHead carries the upstream response metadata known before the body is
// consumed.
type StreamHead struct {
	StatusCode         int
	ContentType        string
	ContentLength      int64
	Location           string
	ContentDisposition string
	IsBinary           bool
}

// UpstreamStream is an open upstream response body with its head metadata.
// Callers must fully consume Body (or close it) and must not use Head after
// an upstream redirect decision changes.
type UpstreamStream struct {
	Head StreamHead
	Body io.ReadCloser
}

type Client struct {
	httpClient *http.Client
}

func NewClient(headerTimeout time.Duration) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// No overall timeout: bodies stream (potentially gigabytes) and only
	// stalled response headers fail fast. Body progress is bounded by the
	// caller's context cancellation.
	transport.ResponseHeaderTimeout = headerTimeout
	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			// Do not follow redirects: the gateway relays the upstream 3xx
			// status together with its Location header to the caller.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// ExecuteStream performs the validated request and returns the open upstream
// response. The caller streams Body in chunks with bounded memory.
func (c *Client) ExecuteStream(ctx context.Context, req domain.ValidatedRequest) (*UpstreamStream, error) {
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.GeneratedURL, bytes.NewReader(req.BodyBytes))
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", req.Body.ContentType)
	}
	if strings.TrimSpace(req.BearerToken) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.BearerToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	disposition := strings.TrimSpace(resp.Header.Get("Content-Disposition"))
	if disposition == "" {
		disposition = syntheticDisposition(req.GeneratedURL)
	}

	return &UpstreamStream{
		Head: StreamHead{
			StatusCode:         resp.StatusCode,
			ContentType:        contentType,
			ContentLength:      resp.ContentLength,
			Location:           resp.Header.Get("Location"),
			ContentDisposition: disposition,
			IsBinary:           isBinaryPayload(contentType, nil),
		},
		Body: resp.Body,
	}, nil
}

// syntheticDisposition derives a Content-Disposition from the file name in
// the request URL, so proxied downloads keep the original name instead of a
// generic one derived from the gateway endpoint path. URLs without a
// file-like last segment (no extension) yield no header, leaving inline
// rendering untouched. An upstream header always wins over synthesis.
func syntheticDisposition(generatedURL string) string {
	name := urlFileName(generatedURL)
	if name == "" {
		return ""
	}
	if isASCIIPrintable(name) {
		return `inline; filename="` + name + `"`
	}
	return `inline; filename="download"; filename*=UTF-8''` + encodeRFC5987(name)
}

// urlFileName extracts a file-like last path segment from a URL.
// It returns "" when the path has no file name (empty, trailing slash) or no
// extension, so API-style paths such as /search stay header-free.
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

func isASCIIPrintable(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7e || value[i] == '"' || value[i] == '\\' {
			return false
		}
	}
	return len(value) > 0
}

// encodeRFC5987 percent-encodes a value per RFC 5987 attr-char rules.
func encodeRFC5987(value string) string {
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			strings.IndexByte("!#$&+-.^_`|~", c) >= 0 {
			builder.WriteByte(c)
		} else {
			fmt.Fprintf(&builder, "%%%02X", c)
		}
	}
	return builder.String()
}

func isBinaryPayload(contentType string, payload []byte) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(ct, "text/") || ct == "application/json" || ct == "application/xml" || strings.HasSuffix(ct, "+json") || strings.HasSuffix(ct, "+xml") {
		return false
	}
	if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/") || ct == "application/octet-stream" || ct == "application/pdf" {
		return true
	}

	return http.DetectContentType(payload) == "application/octet-stream"
}
