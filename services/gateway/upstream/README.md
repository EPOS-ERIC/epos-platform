# upstream client

Cached outbound HTTP client for gateway integrations.

## What consumers get

- First request for unique key (`method + normalized URL + headers + body hash`) goes to network.
- Cold miss (no cache entry at all) waits for real upstream result (or caller context cancellation), capped by `RealTimeout`.
- Successful `2xx` response cached in memory for `CacheTTL`.
- Request inside TTL returns cached response (`X-EPOS-Cache-Hit: true`, `X-EPOS-Cache-Source: memory`).
- If TTL expired, client starts refresh request.
- If caller timeout hits before refresh finishes **and** stale entry exists, stale response returned immediately (`X-EPOS-Cache-Stale: true`, `X-EPOS-Cache-Reason: timeout`). Refresh keeps running in background.
- Concurrent identical requests share one upstream call (in-flight deduplication).

## Cache duration and invalidation

- `CacheTTL > 0`: keep entries until expiry.
- `CacheTTL = 0`: no persistence; cache entry removed after fetch.
- `MaxEntries`: LRU limit (oldest recently used evicted when limit exceeded).
- Entry deleted on refresh failure: network error, response body read error, or non-`2xx` status.
- After deletion, next request behaves like cold miss.

## Timeouts

- `Timeout`: caller-facing wait budget for requests that already have a stale cache candidate.
- `RequestOptions.Timeout`: per-request override.
- `RealTimeout`: upstream budget for actual fetch/refresh (background included). Cold misses are capped by this value. If zero, uses `Timeout`.

## Minimal use

```go
client, err := upstream.NewClient(upstream.Config{
    Timeout:     300 * time.Millisecond,
    RealTimeout: 2 * time.Second,
    CacheTTL:    30 * time.Second,
    MaxEntries:  256,
})
if err != nil {
    return err
}

req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := client.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()

// Behavior flags:
_ = resp.Cache.Hit   // true when served from memory
_ = resp.Cache.Stale // true when stale fallback used
```

## Behavior examples

- Warm cache: `miss -> network 200 -> cached`.
- Fast repeat: `hit -> memory`.
- Cold miss + slow upstream: `wait (up to RealTimeout) -> fresh response or timeout error`.
- Expired + slow upstream: `timeout -> stale returned -> background refresh updates cache if 2xx`.
- Expired + failed refresh: `stale may be returned once during timeout path, then entry deleted; next call is cold miss`.
