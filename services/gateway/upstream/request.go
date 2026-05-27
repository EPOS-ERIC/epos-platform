package upstream

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	nethttp "net/http"
)

type preparedRequest struct {
	key     string
	method  string
	url     string
	header  nethttp.Header
	body    []byte
	hasBody bool
}

func prepareRequest(req *nethttp.Request) (preparedRequest, error) {
	body, hasBody, err := snapshotRequestBody(req)
	if err != nil {
		return preparedRequest{}, fmt.Errorf("snapshot request body: %w", err)
	}

	key, err := buildCacheKey(req, body)
	if err != nil {
		return preparedRequest{}, fmt.Errorf("build cache key: %w", err)
	}

	return preparedRequest{
		key:     key,
		method:  req.Method,
		url:     normalizeURL(req.URL),
		header:  req.Header.Clone(),
		body:    body,
		hasBody: hasBody,
	}, nil
}

func snapshotRequestBody(req *nethttp.Request) ([]byte, bool, error) {
	if req.Body == nil {
		return nil, false, nil
	}

	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, true, fmt.Errorf("clone request body: %w", err)
		}
		defer body.Close()

		data, err := io.ReadAll(body)
		if err != nil {
			return nil, true, fmt.Errorf("read cloned request body: %w", err)
		}

		return data, true, nil
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, true, fmt.Errorf("read request body: %w", err)
	}

	if err := req.Body.Close(); err != nil {
		return nil, true, fmt.Errorf("close request body: %w", err)
	}

	req.Body = io.NopCloser(bytes.NewReader(data))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	req.ContentLength = int64(len(data))

	return data, true, nil
}

func buildCacheKey(req *nethttp.Request, body []byte) (string, error) {
	if req.URL == nil {
		return "", fmt.Errorf("request URL is required")
	}

	var builder strings.Builder
	builder.WriteString(req.Method)
	builder.WriteByte('|')
	builder.WriteString(normalizeURL(req.URL))
	builder.WriteByte('|')
	builder.WriteString(normalizeHeaders(req.Header))
	builder.WriteByte('|')

	if len(body) > 0 {
		hash := sha256.Sum256(body)
		builder.WriteString(hex.EncodeToString(hash[:]))
	}

	return builder.String(), nil
}

func normalizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	clone := *u
	query := clone.Query()
	clone.RawQuery = query.Encode()
	clone.Fragment = ""

	return clone.String()
}

func normalizeHeaders(header nethttp.Header) string {
	if len(header) == 0 {
		return ""
	}

	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		values := append([]string(nil), header.Values(key)...)
		sort.Strings(values)
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(strings.Join(values, ","))
		builder.WriteByte(';')
	}

	return builder.String()
}

func (p preparedRequest) newRequest(ctx context.Context) (*nethttp.Request, error) {
	var body io.Reader
	if p.hasBody {
		body = bytes.NewReader(p.body)
	}

	req, err := nethttp.NewRequestWithContext(ctx, p.method, p.url, body)
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}

	req.Header = p.header.Clone()
	if p.hasBody {
		req.ContentLength = int64(len(p.body))
	}

	return req, nil
}
