/*
Package upstream provides outbound HTTP client helpers for gateway integrations.

It wraps standard net/http behavior with caller-facing request timeouts,
separate real upstream timeouts for background refreshes, in-memory TTL cache
with LRU eviction, stale-cache fallback when caller timeout is reached, and
in-flight request deduplication for identical requests.

Only successful 2xx responses are cached. If a request times out for caller but a
stale cache entry exists, package returns stale entry immediately and lets real
upstream request continue in background. When background request finishes, cache
is updated on 2xx or deleted on error/non-2xx.

Example usage:

	client, err := upstream.NewClient(upstream.Config{
		Timeout:     500 * time.Millisecond,
		RealTimeout: 5 * time.Second,
		CacheTTL:    30 * time.Second,
		MaxEntries:  256,
	})
	if err != nil {
		return err
	}

	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, upstreamURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.Cache.Stale {
		log.Warn("served stale upstream response")
	}

	standard := client.StandardClient()
	_ = standard
*/
package upstream
