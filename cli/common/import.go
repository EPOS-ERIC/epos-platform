package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/cli/display"
)

var importClient = &http.Client{Timeout: 5 * time.Minute}

const maxTurtleBytes = 10 << 20

var exampleURLs = []string{
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/example-geojson.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/example-covjson.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/example-downloadablefile.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/ogc-wms.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/ogc-wfs.ttl",
}

// PopulateTurtle prepares and imports inputs using the configured authentication.
func PopulateTurtle(inputs []string, examples bool, bearerToken string, managedAAI bool, managedToken func() (string, error), apiURL string) ([]string, error) {
	payload, paths, err := PrepareTurtleImport(inputs, examples)
	if err != nil {
		return nil, err
	}

	token := strings.TrimSpace(bearerToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("EPOS_BEARER_TOKEN"))
	}

	if token == "" {
		if !managedAAI {
			return nil, fmt.Errorf("EPOS_BEARER_TOKEN is required for external AAI")
		}

		token, err = managedToken()
		if err != nil {
			return nil, err
		}
	}

	if err := SendTurtleImport(apiURL, token, payload); err != nil {
		return nil, err
	}

	return paths, nil
}

// PrepareTurtleImport collects and combines inputs into a Turtle payload.
func PrepareTurtleImport(inputs []string, examples bool) ([]byte, []string, error) {
	seen := make(map[string]bool)
	var paths []string
	for _, input := range inputs {
		abs, err := filepath.Abs(input)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve input path: %w", err)
		}

		abs, err = filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve input path %q: %w", input, err)
		}

		err = filepath.WalkDir(abs, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if !entry.IsDir() && filepath.Ext(path) == ".ttl" {
				resolved, err := filepath.EvalSymlinks(path)
				if err != nil {
					return err
				}

				if !seen[resolved] {
					seen[resolved] = true
					paths = append(paths, resolved)
				}
			}

			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("collect Turtle files from %q: %w", input, err)
		}
	}

	slices.Sort(paths)
	if examples {
		paths = append(paths, exampleURLs...)
	}

	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no Turtle files found")
	}

	var payload bytes.Buffer
	for _, path := range paths {
		if err := appendTurtle(&payload, path); err != nil {
			return nil, nil, err
		}
	}

	return payload.Bytes(), paths, nil
}

func appendTurtle(dst *bytes.Buffer, path string) error {
	var src io.ReadCloser
	if strings.HasPrefix(path, "https://") {
		res, err := importClient.Get(path)
		if err != nil {
			return fmt.Errorf("download example %q: %w", path, err)
		}

		if res.StatusCode != http.StatusOK {
			if err := res.Body.Close(); err != nil {
				return fmt.Errorf("close example response: %w", err)
			}

			return fmt.Errorf("download example %q: HTTP status %d", path, res.StatusCode)
		}

		src = res.Body
	} else {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open Turtle file %q: %w", path, err)
		}

		src = file
	}
	defer src.Close()

	pos := int64(dst.Len())
	n, err := io.Copy(dst, io.LimitReader(src, maxTurtleBytes-pos+1))
	if err != nil {
		return fmt.Errorf("read Turtle input %q: %w", path, err)
	}

	if pos+n+1 > maxTurtleBytes {
		return fmt.Errorf("combined Turtle inputs exceed 10 MiB import limit")
	}

	if _, err := dst.WriteString("\n"); err != nil {
		return fmt.Errorf("write Turtle separator: %w", err)
	}

	return nil
}

// RequestManagedToken obtains a managed AAI administrator token.
func RequestManagedToken(root, email, password string) (string, error) {
	endpoint, err := url.JoinPath(root, "oauth/token")
	if err != nil {
		return "", fmt.Errorf("build AAI token URL: %w", err)
	}

	form := url.Values{
		"grant_type": {"password"},
		"username":   {email},
		"password":   {password},
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create AAI token request: %w", err)
	}

	req.SetBasicAuth("local-dev-client", "dev-secret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := *importClient
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request AAI token: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AAI token request failed with HTTP status %d; set EPOS_BEARER_TOKEN if administrator credentials have changed", res.StatusCode)
	}

	var response struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&response); err != nil {
		return "", fmt.Errorf("decode AAI token response: %w", err)
	}

	if response.AccessToken == "" {
		return "", fmt.Errorf("AAI token response did not contain an access token")
	}

	return response.AccessToken, nil
}

// SendTurtleImport uploads a prepared Turtle payload and displays the result.
func SendTurtleImport(baseURL, token string, payload []byte) error {
	endpoint, err := url.JoinPath(baseURL, "imports/turtle")
	if err != nil {
		return fmt.Errorf("build import URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, io.NopCloser(bytes.NewReader(payload)))
	if err != nil {
		return fmt.Errorf("create import request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/turtle")

	res, err := importClient.Do(req)
	if err != nil {
		return fmt.Errorf("send import request; outcome may be unknown, check the environment before retrying: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		mediaType, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
		if err == nil && mediaType == "application/problem+json" {
			var problem struct {
				Type     string `json:"type"`
				Title    string `json:"title"`
				Detail   string `json:"detail"`
				Target   string `json:"target"`
				Instance string `json:"instance"`
			}
			if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&problem); err == nil {
				parts := []string{fmt.Sprintf("import request failed with HTTP status %d", res.StatusCode)}
				for _, item := range []struct{ label, value string }{
					{"type", problem.Type},
					{"title", problem.Title},
					{"detail", problem.Detail},
					{"target", problem.Target},
					{"trace", problem.Instance},
				} {
					if value := strings.TrimSpace(item.value); value != "" {
						parts = append(parts, item.label+": "+value)
					}
				}
				return fmt.Errorf("%s", strings.Join(parts, "; "))
			}
		}
		return fmt.Errorf("import request failed with HTTP status %d", res.StatusCode)
	}

	var result struct {
		Groups               int `json:"groups"`
		Categories           int `json:"categories"`
		Organizations        int `json:"organizations"`
		ContactPoints        int `json:"contact_points"`
		Datasets             int `json:"datasets"`
		SoftwareApplications int `json:"software_applications"`
		SoftwareSourceCodes  int `json:"software_source_codes"`
		Warnings             []struct {
			Severity  string `json:"severity"`
			Subject   string `json:"subject"`
			Predicate string `json:"predicate"`
			Message   string `json:"message"`
		} `json:"warnings"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("import succeeded but response could not be read; do not retry without checking the environment: %w", err)
	}

	display.Info("Import counts: groups=%d categories=%d organizations=%d contact_points=%d datasets=%d software_applications=%d software_source_codes=%d", result.Groups, result.Categories, result.Organizations, result.ContactPoints, result.Datasets, result.SoftwareApplications, result.SoftwareSourceCodes)

	for _, warning := range result.Warnings {
		display.Warn("Import %s: subject=%s predicate=%s: %s", warning.Severity, warning.Subject, warning.Predicate, warning.Message)
	}

	return nil
}
