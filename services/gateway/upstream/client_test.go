package upstream

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	statusOK          = "200 OK"
	contentTypeHeader = "Content-Type"
	jsonContentType   = "application/json"
)

func TestClientReturnsValidCachedEntry(t *testing.T) {
	transport := newStubTransport()
	transport.queueResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"fresh"}`)),
	})

	client := newTestClient(t, transport, time.Second, time.Minute)
	request := mustNewRequest(t, "GET", "http://metadata-lifecycle:8080/api/v1/resources?q=b&a=1", nil)

	first, err := client.Do(request)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	if got := readBody(t, first.Response); got != `{"value":"fresh"}` {
		t.Fatalf("unexpected first body: %s", got)
	}

	second, err := client.Do(request)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	if got := readBody(t, second.Response); got != `{"value":"fresh"}` {
		t.Fatalf("unexpected cached body: %s", got)
	}

	if !second.Cache.Hit || second.Cache.Stale {
		t.Fatalf("expected fresh cache hit, got %+v", second.Cache)
	}

	if got := second.Header.Get(cacheSourceHeader); got != cacheSourceMemory {
		t.Fatalf("unexpected cache source header: %q", got)
	}

	if got := transport.callCount(); got != 1 {
		t.Fatalf("unexpected transport call count: got %d want 1", got)
	}
}

func TestClientReturnsStaleOnTimeoutAndRefreshesCache(t *testing.T) {
	transport := newStubTransport()
	transport.queueResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"old"}`)),
	})

	client := newTestClient(t, transport, time.Second, time.Minute)
	now := time.Unix(100, 0)
	client.now = func() time.Time { return now }

	request := mustNewRequest(t, "GET", "http://metadata-lifecycle:8080/api/v1/resources", nil)
	if _, err := client.Do(request); err != nil {
		t.Fatalf("warm request failed: %v", err)
	}

	now = now.Add(2 * time.Minute)
	control := transport.queueBlockedResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"new"}`)),
	})

	stale, err := client.DoWithOptions(request, RequestOptions{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("stale fallback failed: %v", err)
	}

	<-control.start

	if got := readBody(t, stale.Response); got != `{"value":"old"}` {
		t.Fatalf("unexpected stale body: %s", got)
	}

	if !stale.Cache.Hit || !stale.Cache.Stale || stale.Cache.Reason != cacheReasonTimeout {
		t.Fatalf("unexpected stale metadata: %+v", stale.Cache)
	}

	close(control.release)
	transport.waitIdle()

	fresh, err := client.Do(request)
	if err != nil {
		t.Fatalf("refreshed request failed: %v", err)
	}

	if got := readBody(t, fresh.Response); got != `{"value":"new"}` {
		t.Fatalf("unexpected refreshed body: %s", got)
	}
}

func TestClientDeletesStaleEntryWhenRefreshFails(t *testing.T) {
	transport := newStubTransport()
	transport.queueResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"old"}`)),
	})

	client := newTestClient(t, transport, time.Second, time.Minute)
	now := time.Unix(100, 0)
	client.now = func() time.Time { return now }

	request := mustNewRequest(t, "GET", "http://metadata-lifecycle:8080/api/v1/resources", nil)
	if _, err := client.Do(request); err != nil {
		t.Fatalf("warm request failed: %v", err)
	}

	now = now.Add(2 * time.Minute)
	control := transport.queueBlockedResponse(&http.Response{
		Status:     "500 Internal Server Error",
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":"broken"}`)),
	})

	stale, err := client.DoWithOptions(request, RequestOptions{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("stale fallback failed: %v", err)
	}

	<-control.start
	if !stale.Cache.Stale {
		t.Fatalf("expected stale cache response")
	}

	close(control.release)
	transport.waitIdle()
	waitForNoInflight(t, client)

	control = transport.queueBlockedResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"next"}`)),
	})

	_, err = client.DoWithOptions(request, RequestOptions{Timeout: time.Millisecond})
	if err == nil {
		t.Fatal("expected timeout without stale cache")
	}

	<-control.start
	close(control.release)
	transport.waitIdle()
}

func TestClientDeduplicatesInflightRequests(t *testing.T) {
	transport := newStubTransport()
	control := transport.queueBlockedResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"shared"}`)),
	})

	client := newTestClient(t, transport, time.Second, time.Minute)
	request := mustNewRequest(t, "GET", "http://metadata-lifecycle:8080/api/v1/resources", nil)

	var wg sync.WaitGroup
	results := make(chan string, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Do(request)
			if err != nil {
				results <- "error"
				return
			}
			results <- readBody(t, resp.Response)
		}()
	}

	<-control.start
	close(control.release)
	wg.Wait()
	close(results)

	for result := range results {
		if result != `{"value":"shared"}` {
			t.Fatalf("unexpected concurrent result: %s", result)
		}
	}

	if got := transport.callCount(); got != 1 {
		t.Fatalf("unexpected transport call count: got %d want 1", got)
	}
}

func TestClientReturnsStaleImmediatelyWhileRefreshContinues(t *testing.T) {
	transport := newStubTransport()
	transport.queueResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"old"}`)),
	})

	client := newTestClient(t, transport, 5*time.Millisecond, time.Minute)
	now := time.Unix(100, 0)
	client.now = func() time.Time { return now }

	request := mustNewRequest(t, "GET", "http://metadata-lifecycle:8080/api/v1/resources", nil)
	if _, err := client.Do(request); err != nil {
		t.Fatalf("warm request failed: %v", err)
	}

	now = now.Add(2 * time.Minute)
	control := transport.queueBlockedResponse(&http.Response{
		Status:     statusOK,
		StatusCode: http.StatusOK,
		Header:     http.Header{contentTypeHeader: {jsonContentType}},
		Body:       io.NopCloser(strings.NewReader(`{"value":"new"}`)),
	})

	first, err := client.DoWithOptions(request, RequestOptions{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("first stale fallback failed: %v", err)
	}
	<-control.start

	if !first.Cache.Stale {
		t.Fatalf("expected stale first response")
	}

	second, err := client.Do(request)
	if err != nil {
		t.Fatalf("second stale response failed: %v", err)
	}

	if got := readBody(t, second.Response); got != `{"value":"old"}` {
		t.Fatalf("unexpected repeated stale body: %s", got)
	}

	if !second.Cache.Stale {
		t.Fatalf("expected stale second response")
	}

	close(control.release)
	transport.waitIdle()
}

func newTestClient(t *testing.T, transport *stubTransport, timeout, ttl time.Duration) *Client {
	t.Helper()

	client, err := NewClient(Config{
		Timeout:     timeout,
		RealTimeout: time.Second,
		CacheTTL:    ttl,
		Transport:   transport,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	return client
}

func mustNewRequest(t *testing.T, method, rawURL string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), method, rawURL, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	return req
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(body)
}

func waitForNoInflight(t *testing.T, client *Client) {
	t.Helper()

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		client.mu.Lock()
		count := len(client.inflight)
		client.mu.Unlock()
		if count == 0 {
			return
		}

		select {
		case <-deadline.C:
			t.Fatal("timed out waiting for inflight cleanup")
		case <-ticker.C:
		}
	}
}

type stubTransport struct {
	mu      sync.Mutex
	plans   []*responsePlan
	active  sync.WaitGroup
	callNum int
}

type responsePlan struct {
	response *http.Response
	start    chan struct{}
	release  chan struct{}
	blocked  bool
}

type blockedResponseControl struct {
	start   <-chan struct{}
	release chan struct{}
}

func newStubTransport() *stubTransport {
	return &stubTransport{}
}

func (s *stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	s.active.Add(1)
	defer s.active.Done()

	s.mu.Lock()
	if len(s.plans) == 0 {
		s.mu.Unlock()
		return nil, context.DeadlineExceeded
	}

	plan := s.plans[0]
	s.plans = s.plans[1:]
	s.callNum++
	s.mu.Unlock()

	if plan.start != nil {
		close(plan.start)
	}

	if plan.blocked {
		<-plan.release
	}

	return cloneHTTPResponse(plan.response), nil
}

func (s *stubTransport) queueResponse(response *http.Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans = append(s.plans, &responsePlan{response: response})
}

func (s *stubTransport) queueBlockedResponse(response *http.Response) blockedResponseControl {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := make(chan struct{})
	release := make(chan struct{})
	s.plans = append(s.plans, &responsePlan{
		response: response,
		start:    start,
		release:  release,
		blocked:  true,
	})

	return blockedResponseControl{start: start, release: release}
}

func (s *stubTransport) waitIdle() {
	s.active.Wait()
}

func (s *stubTransport) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callNum
}

func cloneHTTPResponse(resp *http.Response) *http.Response {
	if resp == nil {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(strings.NewReader(string(body)))

	return &http.Response{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Proto:      resp.Proto,
		ProtoMajor: resp.ProtoMajor,
		ProtoMinor: resp.ProtoMinor,
	}
}
