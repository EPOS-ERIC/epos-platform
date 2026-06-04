package testfixtures

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

var gatewayRoot = mustGatewayRoot()

func mustGatewayRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve test fixture root")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func Path(relativePath string) string {
	return filepath.Join(gatewayRoot, filepath.FromSlash(relativePath))
}

func Read(t testing.TB, relativePath string) []byte {
	t.Helper()

	data, err := os.ReadFile(Path(relativePath))
	if err != nil {
		t.Fatalf("read fixture %s: %v", relativePath, err)
	}

	return data
}

func DecodeJSON(t testing.TB, relativePath string, target any) {
	t.Helper()

	if err := json.Unmarshal(Read(t, relativePath), target); err != nil {
		t.Fatalf("decode fixture %s: %v", relativePath, err)
	}
}

func LoadAccessDetail(t testing.TB, relativePath string) *metadataclient.AccessDetailResponse {
	t.Helper()

	var detail metadataclient.AccessDetailResponse
	DecodeJSON(t, relativePath, &detail)
	return &detail
}

func NewMetadataClient(baseURL string) *metadataclient.APIClient {
	config := metadataclient.NewConfiguration()
	config.Servers = metadataclient.ServerConfigurations{{URL: baseURL, Description: "", Variables: nil}}
	return metadataclient.NewAPIClient(config)
}

func NewMetadataServer(t testing.TB, routes map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixturePath, ok := routes[r.URL.RequestURI()]
		if !ok {
			fixturePath, ok = routes[r.URL.Path]
		}
		if !ok {
			fixturePath, ok = routes[strings.ReplaceAll(r.URL.Path, ":", "%3A")]
		}
		if !ok {
			requestPath, err := url.PathUnescape(r.URL.Path)
			if err == nil {
				for routePath, candidate := range routes {
					normalizedRoutePath, normalizeErr := url.PathUnescape(routePath)
					if normalizeErr == nil && normalizedRoutePath == requestPath {
						fixturePath = candidate
						ok = true
						break
					}
				}
			}
		}
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(Read(t, fixturePath)); err != nil {
			t.Fatalf("write fixture response %s: %v", fixturePath, err)
		}
	}))
}
