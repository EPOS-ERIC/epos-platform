package upstream

import (
	"bytes"
	"fmt"
	"io"

	nethttp "net/http"
)

const (
	cacheHitHeader     = "X-EPOS-Cache-Hit"
	cacheStaleHeader   = "X-EPOS-Cache-Stale"
	cacheReasonHeader  = "X-EPOS-Cache-Reason"
	cacheSourceHeader  = "X-EPOS-Cache-Source"
	cacheSourceFresh   = "network"
	cacheSourceMemory  = "memory"
	cacheReasonTimeout = CacheReason("timeout")
	cacheReasonNone    = CacheReason("")
)

// CacheReason describes why a cached response was returned.
type CacheReason string

// CacheMetadata describes whether a response came from cache.
type CacheMetadata struct {
	// Hit is true when response was served from in-memory cache.
	Hit bool
	// Stale is true when expired cache was served after caller timeout.
	Stale bool
	// Reason explains cache serving cause (for example timeout fallback).
	Reason CacheReason
}

// Response wraps a standard HTTP response with cache metadata.
type Response struct {
	// Response is the standard net/http response payload.
	*nethttp.Response
	// Cache contains cache-related metadata for this response.
	Cache CacheMetadata
}

type cachedResponse struct {
	status        string
	statusCode    int
	header        nethttp.Header
	body          []byte
	contentLength int64
	proto         string
	protoMajor    int
	protoMinor    int
}

func staleResponse(cached *cachedResponse) *Response {
	return buildResponse(cached, CacheMetadata{
		Hit:    true,
		Stale:  true,
		Reason: cacheReasonTimeout,
	})
}

func newCachedResponse(resp *nethttp.Response, body []byte) *cachedResponse {
	return &cachedResponse{
		status:        resp.Status,
		statusCode:    resp.StatusCode,
		header:        resp.Header.Clone(),
		body:          bytes.Clone(body),
		contentLength: int64(len(body)),
		proto:         resp.Proto,
		protoMajor:    resp.ProtoMajor,
		protoMinor:    resp.ProtoMinor,
	}
}

func readResponseBody(resp *nethttp.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return body, nil
}

func buildResponse(cached *cachedResponse, meta CacheMetadata) *Response {
	header := cached.header.Clone()
	header.Set(cacheSourceHeader, cacheSourceFresh)
	if meta.Hit {
		header.Set(cacheHitHeader, "true")
		header.Set(cacheSourceHeader, cacheSourceMemory)
	} else {
		header.Set(cacheHitHeader, "false")
	}

	if meta.Stale {
		header.Set(cacheStaleHeader, "true")
	} else {
		header.Set(cacheStaleHeader, "false")
	}

	if meta.Reason != cacheReasonNone {
		header.Set(cacheReasonHeader, string(meta.Reason))
	} else {
		header.Del(cacheReasonHeader)
	}

	return &Response{
		Response: &nethttp.Response{
			Status:        cached.status,
			StatusCode:    cached.statusCode,
			Header:        header,
			Body:          io.NopCloser(bytes.NewReader(cached.body)),
			ContentLength: cached.contentLength,
			Proto:         cached.proto,
			ProtoMajor:    cached.protoMajor,
			ProtoMinor:    cached.protoMinor,
		},
		Cache: meta,
	}
}
