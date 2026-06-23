package upstream

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	nethttp "net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Config configures a cached upstream HTTP client.
type Config struct {
	// Service identifies upstream dependency name for tracing.
	Service string
	// Timeout is caller-facing max wait time per request.
	Timeout time.Duration
	// RealTimeout is max upstream time for background refresh requests.
	RealTimeout time.Duration
	// CacheTTL is in-memory cache validity duration.
	// Zero disables cache persistence.
	CacheTTL time.Duration
	// MaxEntries is max number of cached responses kept in memory.
	MaxEntries int
	// Transport is optional custom HTTP transport.
	// If nil, net/http.DefaultTransport is used.
	Transport nethttp.RoundTripper
}

// RequestOptions configures one request execution.
type RequestOptions struct {
	// Timeout overrides client Timeout for this request.
	// If zero, client Timeout is used.
	Timeout time.Duration
}

// Client is an outbound HTTP client with in-memory response caching.
type Client struct {
	httpClient  *nethttp.Client
	service     string
	timeout     time.Duration
	realTimeout time.Duration
	cacheTTL    time.Duration
	maxEntries  int
	log         *slog.Logger
	now         func() time.Time

	mu       sync.Mutex
	cache    map[string]*cacheEntry
	order    *list.List
	inflight map[string]*inflightCall
}

type inflightCall struct {
	done     chan struct{}
	response *cachedResponse
	err      error
	staleMu  sync.Mutex
	stale    bool
}

type upstreamStatusError struct {
	method   string
	url      string
	status   string
	response *cachedResponse
}

func (e *upstreamStatusError) Error() string {
	return fmt.Sprintf("unexpected upstream status for %s %s: %s", e.method, e.url, e.status)
}

// NewClient creates a cached upstream HTTP client from Config.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}

	if cfg.CacheTTL < 0 {
		return nil, fmt.Errorf("cache TTL must be zero or greater")
	}

	if cfg.RealTimeout <= 0 {
		return nil, fmt.Errorf("real timeout must be greater than zero")
	}

	if cfg.MaxEntries <= 0 {
		return nil, fmt.Errorf("max entries must be greater than zero")
	}

	transport := cfg.Transport
	if transport == nil {
		transport = nethttp.DefaultTransport
	}
	transport = otelhttp.NewTransport(transport)

	return &Client{
		httpClient:  &nethttp.Client{Transport: transport, CheckRedirect: nil, Jar: nil, Timeout: 0},
		service:     cfg.Service,
		timeout:     cfg.Timeout,
		realTimeout: cfg.RealTimeout,
		cacheTTL:    cfg.CacheTTL,
		maxEntries:  cfg.MaxEntries,
		log:         logging.Get("gateway-http-client"),
		now:         time.Now,
		mu:          sync.Mutex{},
		cache:       make(map[string]*cacheEntry),
		order:       list.New(),
		inflight:    make(map[string]*inflightCall),
	}, nil
}

// StandardClient returns a standard net/http client that uses this cache-aware transport.
func (c *Client) StandardClient() *nethttp.Client {
	return &nethttp.Client{Transport: c, CheckRedirect: nil, Jar: nil, Timeout: 0}
}

// RoundTrip implements net/http.RoundTripper.
func (c *Client) RoundTrip(req *nethttp.Request) (*nethttp.Response, error) {
	resp, err := c.Do(req)
	if err != nil {
		if statusErr, ok := errors.AsType[*upstreamStatusError](err); ok {
			return buildResponse(statusErr.response, CacheMetadata{Hit: false, Stale: false, Reason: cacheReasonNone}).Response, nil
		}

		return nil, err
	}

	return resp.Response, nil
}

// Do executes req using client defaults.
func (c *Client) Do(req *nethttp.Request) (*Response, error) {
	return c.DoWithOptions(req, RequestOptions{Timeout: 0})
}

// DoWithOptions executes req with optional per-request overrides.
func (c *Client) DoWithOptions(req *nethttp.Request, opts RequestOptions) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	prepared, err := prepareRequest(req)
	if err != nil {
		return nil, err
	}

	timeout := c.timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	return c.doPrepared(req.Context(), prepared, timeout)
}

func (c *Client) doPrepared(ctx context.Context, prepared preparedRequest, timeout time.Duration) (*Response, error) {
	c.annotateSpan(ctx, attribute.String("upstream.service", c.service))
	if c.cacheTTL <= 0 {
		c.annotateSpan(ctx, attribute.String("cache.mode", "disabled"))
	}

	now := c.now()
	if entry := c.getCache(prepared.key); entry != nil && now.Before(entry.expiresAt) {
		c.annotateSpan(ctx, attribute.String("cache.mode", "cache-hit-fresh"))
		c.log.DebugContext(ctx, "upstream request behavior",
			slog.Group("cache", "mode", "cache-hit-fresh"),
			slog.Group("upstream", "method", prepared.method, "url", prepared.url),
		)
		return buildResponse(entry.response, CacheMetadata{Hit: true, Stale: false, Reason: cacheReasonNone}), nil
	}

	staleEntry := c.getCache(prepared.key)
	if staleEntry == nil {
		mode := "cache-miss-cold"
		if c.cacheTTL <= 0 {
			mode = "disabled"
		}
		c.annotateSpan(ctx, attribute.String("cache.mode", mode))
		c.log.DebugContext(ctx, "upstream request behavior",
			slog.Group("cache", "mode", mode),
			slog.Group("upstream", "method", prepared.method, "url", prepared.url),
			slog.Group("timeouts", "real", c.realTimeout),
		)
	} else {
		c.annotateSpan(ctx, attribute.String("cache.mode", "cache-hit-stale-candidate"))
		c.log.DebugContext(ctx, "upstream request behavior",
			slog.Group("cache", "mode", "cache-hit-stale-candidate"),
			slog.Group("upstream", "method", prepared.method, "url", prepared.url),
			slog.Group("timeouts", "caller", timeout, "real", c.realTimeout),
		)
	}

	inflightCtx := ctx
	if staleEntry != nil {
		inflightCtx = context.WithoutCancel(ctx)
	}

	call, existing := c.getOrStartInflight(inflightCtx, prepared.key, prepared)
	if existing && staleEntry != nil && call.isStaleReady() {
		c.annotateSpan(ctx, attribute.String("cache.mode", "stale-served-inflight"))
		c.log.DebugContext(ctx, "upstream request behavior",
			slog.Group("cache", "mode", "stale-served-inflight"),
			slog.Group("upstream", "method", prepared.method, "url", prepared.url),
		)
		return staleResponse(staleEntry.response), nil
	}

	if staleEntry == nil {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for inflight request: %w", ctx.Err())
		case <-call.done:
			return c.finishForeground(call)
		}
	}

	if timeout <= 0 {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for inflight request without timeout: %w", ctx.Err())
		case <-call.done:
			return c.finishForeground(call)
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for inflight request with timeout: %w", ctx.Err())
	case <-call.done:
		return c.finishForeground(call)
	case <-timer.C:
		call.markStaleReady()
		c.annotateSpan(ctx, attribute.String("cache.mode", "stale-served-timeout"))
		c.log.DebugContext(ctx, "upstream request behavior",
			slog.Group("cache", "mode", "stale-served-timeout"),
			slog.Group("upstream", "method", prepared.method, "url", prepared.url),
			slog.Group("timeouts", "caller", timeout),
		)
		return staleResponse(staleEntry.response), nil
	}
}

func (c *Client) annotateSpan(ctx context.Context, attrs ...attribute.KeyValue) {
	if len(attrs) == 0 {
		return
	}

	trace.SpanFromContext(ctx).SetAttributes(attrs...)
}

func (c *Client) finishForeground(call *inflightCall) (*Response, error) {
	if call.err != nil {
		return nil, call.err
	}

	return buildResponse(call.response, CacheMetadata{Hit: false, Stale: false, Reason: cacheReasonNone}), nil
}

func (c *Client) getOrStartInflight(ctx context.Context, key string, prepared preparedRequest) (*inflightCall, bool) {
	c.mu.Lock()
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		c.log.DebugContext(ctx, "upstream inflight joined", slog.Group("upstream", "method", prepared.method, "url", prepared.url))
		return call, true
	}

	call := &inflightCall{done: make(chan struct{}), response: nil, err: nil, staleMu: sync.Mutex{}, stale: false}
	c.inflight[key] = call
	c.mu.Unlock()

	go c.execute(ctx, key, prepared, call)

	return call, false
}

func (c *Client) execute(ctx context.Context, key string, prepared preparedRequest, call *inflightCall) {
	defer close(call.done)
	defer c.clearInflight(key)

	requestCtx, cancel := context.WithTimeout(ctx, c.realTimeout)
	defer cancel()

	req, err := prepared.newRequest(requestCtx)
	if err != nil {
		call.err = err
		c.deleteCache(key)
		return
	}

	cached, err := c.fetch(requestCtx, prepared, req)
	if err != nil {
		call.err = err
		return
	}

	if c.cacheTTL > 0 {
		c.storeCache(key, cached)
		c.log.DebugContext(requestCtx, "upstream cache stored",
			slog.Group("upstream", "method", prepared.method, "url", prepared.url),
			slog.Group("cache", "ttl", c.cacheTTL),
		)
	} else {
		c.deleteCache(key)
		c.log.DebugContext(requestCtx, "upstream cache disabled", slog.Group("upstream", "method", prepared.method, "url", prepared.url))
	}

	call.response = cached
}

func (c *Client) fetch(ctx context.Context, prepared preparedRequest, req *nethttp.Request) (*cachedResponse, error) {
	c.log.DebugContext(ctx, "upstream request", slog.Group("upstream", "method", prepared.method, "url", prepared.url))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.deleteCache(prepared.key)
		c.log.WarnContext(ctx, "upstream request failed", slog.Group("upstream", "method", prepared.method, "url", prepared.url), "error", err)
		return nil, fmt.Errorf("upstream request failed: %s %s: %w", prepared.method, prepared.url, err)
	}
	defer resp.Body.Close()

	return c.cacheableResponse(ctx, prepared, resp)
}

func (c *Client) cacheableResponse(ctx context.Context, prepared preparedRequest, resp *nethttp.Response) (*cachedResponse, error) {
	body, err := readResponseBody(resp)
	if err != nil {
		c.deleteCache(prepared.key)
		c.log.WarnContext(ctx, "upstream response read failed", slog.Group("upstream", "method", prepared.method, "url", prepared.url), "error", err)
		return nil, err
	}

	if resp.StatusCode < nethttp.StatusOK || resp.StatusCode >= nethttp.StatusMultipleChoices {
		c.deleteCache(prepared.key)
		c.log.WarnContext(ctx, "upstream returned non-cacheable status",
			slog.Group("upstream", "method", prepared.method, "url", prepared.url),
			slog.Group("response", "status", resp.StatusCode),
		)
		return nil, &upstreamStatusError{
			method:   prepared.method,
			url:      prepared.url,
			status:   resp.Status,
			response: newCachedResponse(resp, body),
		}
	}

	return newCachedResponse(resp, body), nil
}

func (c *Client) clearInflight(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.inflight, key)
}

func (c *inflightCall) markStaleReady() {
	c.staleMu.Lock()
	defer c.staleMu.Unlock()
	c.stale = true
}

func (c *inflightCall) isStaleReady() bool {
	c.staleMu.Lock()
	defer c.staleMu.Unlock()
	return c.stale
}
