package upstream

import (
	"container/list"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	nethttp "net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
)

const defaultMaxEntries = 128

// Config configures a cached upstream HTTP client.
type Config struct {
	// Timeout is caller-facing max wait time per request.
	Timeout time.Duration
	// RealTimeout is max upstream time for background refresh requests.
	// If zero, Timeout is used.
	RealTimeout time.Duration
	// CacheTTL is in-memory cache validity duration.
	// Zero disables cache persistence.
	CacheTTL time.Duration
	// MaxEntries is max number of cached responses kept in memory.
	// If zero or less, a safe default is used.
	MaxEntries int
	// Transport is optional custom HTTP transport.
	// If nil, net/http.DefaultTransport is used.
	Transport nethttp.RoundTripper
	// Logger is optional structured logger used for failure and eviction logs.
	// If nil, package default logger is used.
	Logger *slog.Logger
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

// NewClient creates a cached upstream HTTP client from Config.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}

	if cfg.CacheTTL < 0 {
		return nil, fmt.Errorf("cache TTL must be zero or greater")
	}

	realTimeout := cfg.RealTimeout
	if realTimeout <= 0 {
		realTimeout = cfg.Timeout
	}

	maxEntries := cfg.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}

	transport := cfg.Transport
	if transport == nil {
		transport = nethttp.DefaultTransport
	}

	logger := cfg.Logger
	if logger == nil {
		logger = logging.Get("gateway-http-client")
	}

	return &Client{
		httpClient:  &nethttp.Client{Transport: transport},
		timeout:     cfg.Timeout,
		realTimeout: realTimeout,
		cacheTTL:    cfg.CacheTTL,
		maxEntries:  maxEntries,
		log:         logger,
		now:         time.Now,
		cache:       make(map[string]*cacheEntry),
		order:       list.New(),
		inflight:    make(map[string]*inflightCall),
	}, nil
}

// StandardClient returns a standard net/http client that uses this cache-aware transport.
func (c *Client) StandardClient() *nethttp.Client {
	return &nethttp.Client{Transport: c}
}

// RoundTrip implements net/http.RoundTripper.
func (c *Client) RoundTrip(req *nethttp.Request) (*nethttp.Response, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Response, nil
}

// Do executes req using client defaults.
func (c *Client) Do(req *nethttp.Request) (*Response, error) {
	return c.DoWithOptions(req, RequestOptions{})
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
	now := c.now()
	if entry := c.getCache(prepared.key); entry != nil && now.Before(entry.expiresAt) {
		return buildResponse(entry.response, CacheMetadata{Hit: true}), nil
	}

	staleEntry := c.getCache(prepared.key)
	backgroundCtx := context.WithoutCancel(ctx)
	call, existing := c.getOrStartInflight(backgroundCtx, prepared.key, prepared)
	if existing && staleEntry != nil && call.isStaleReady() {
		return staleResponse(staleEntry.response), nil
	}

	if timeout <= 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-call.done:
			return c.finishForeground(call)
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.done:
		return c.finishForeground(call)
	case <-timer.C:
		if staleEntry == nil {
			return nil, fmt.Errorf("request timeout after %s: %w", timeout, context.DeadlineExceeded)
		}

		call.markStaleReady()
		return staleResponse(staleEntry.response), nil
	}
}

func (c *Client) finishForeground(call *inflightCall) (*Response, error) {
	if call.err != nil {
		return nil, call.err
	}

	return buildResponse(call.response, CacheMetadata{}), nil
}

func (c *Client) getOrStartInflight(ctx context.Context, key string, prepared preparedRequest) (*inflightCall, bool) {
	c.mu.Lock()
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		return call, true
	}

	call := &inflightCall{done: make(chan struct{})}
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

	cached, err := c.fetch(prepared, req)
	if err != nil {
		call.err = err
		return
	}

	if c.cacheTTL > 0 {
		c.storeCache(key, cached)
	} else {
		c.deleteCache(key)
	}

	call.response = cached
}

func (c *Client) fetch(prepared preparedRequest, req *nethttp.Request) (*cachedResponse, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.deleteCache(prepared.key)
		c.log.Warn("upstream request failed", "method", prepared.method, "url", prepared.url, "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	return c.cacheableResponse(prepared, resp)
}

func (c *Client) cacheableResponse(prepared preparedRequest, resp *nethttp.Response) (*cachedResponse, error) {
	body, err := readResponseBody(resp)
	if err != nil {
		c.deleteCache(prepared.key)
		c.log.Warn("upstream response read failed", "method", prepared.method, "url", prepared.url, "error", err)
		return nil, err
	}

	if resp.StatusCode < nethttp.StatusOK || resp.StatusCode >= nethttp.StatusMultipleChoices {
		err = fmt.Errorf("unexpected upstream status: %s", resp.Status)
		c.deleteCache(prepared.key)
		c.log.Warn("upstream returned non-cacheable status", "method", prepared.method, "url", prepared.url, "status", resp.StatusCode)
		return nil, err
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
