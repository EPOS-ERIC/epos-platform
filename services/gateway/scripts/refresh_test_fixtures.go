package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	defaultBaseURL = "http://localhost:8080"
	manifestPath   = "testdata/metadata-lifecycle-fixtures.json"
)

type manifest struct {
	Fixtures []fixture `json:"fixtures"`
}

type fixture struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

func main() {
	root, err := gatewayRoot()
	if err != nil {
		fail(err)
	}

	data, err := os.ReadFile(filepath.Join(root, manifestPath))
	if err != nil {
		fail(fmt.Errorf("read manifest: %w", err))
	}

	var cfg manifest
	if err := json.Unmarshal(data, &cfg); err != nil {
		fail(fmt.Errorf("decode manifest: %w", err))
	}

	baseURL := os.Getenv("METADATA_LIFECYCLE_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	client := &http.Client{Timeout: 30 * time.Second}
	for _, fixture := range cfg.Fixtures {
		if err := refreshFixture(client, baseURL, root, fixture); err != nil {
			fail(err)
		}
	}
}

func refreshFixture(client *http.Client, baseURL, root string, fixture fixture) error {
	requestURL, err := resolveURL(baseURL, fixture.URL)
	if err != nil {
		return fmt.Errorf("resolve fixture url %s: %w", fixture.URL, err)
	}

	resp, err := client.Get(requestURL)
	if err != nil {
		return fmt.Errorf("request fixture %s: %w", fixture.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request fixture %s: unexpected status %d", fixture.URL, resp.StatusCode)
	}

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode fixture %s: %w", fixture.URL, err)
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode fixture %s: %w", fixture.URL, err)
	}
	encoded = append(encoded, '\n')

	outputPath := filepath.Join(root, filepath.FromSlash(fixture.Path))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("mkdir fixture dir %s: %w", fixture.Path, err)
	}

	if err := os.WriteFile(outputPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write fixture %s: %w", fixture.Path, err)
	}

	return nil
}

func resolveURL(baseURL, path string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(ref).String(), nil
}

func gatewayRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve gateway root")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..")), nil
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}
