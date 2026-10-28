package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/cli/db"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
)

const maxTurtleBytes = 10 << 20

var populateClient = &http.Client{Timeout: 5 * time.Minute}

var exampleURLs = []string{
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/example-geojson.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/example-covjson.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/example-downloadablefile.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/ogc-wms.ttl",
	"https://raw.githubusercontent.com/EPOS-ERIC/opensource-docs/refs/heads/main/static/examples/ogc-wfs.ttl",
}

// PopulateOpts defines inputs for Populate.
type PopulateOpts struct {
	TTLDirs          []string
	Name             string
	PopulateExamples bool
	BearerToken      string
}

// Populate combines Turtle inputs and imports them in a single request.
func Populate(opts PopulateOpts) (*Env, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid populate parameters: %w", err)
	}
	env, err := GetEnv(opts.Name)
	if err != nil {
		return nil, fmt.Errorf("load docker environment: %w", err)
	}
	paths, err := collectTurtleFiles(opts.TTLDirs)
	if err != nil {
		return nil, err
	}
	if opts.PopulateExamples {
		paths = append(paths, exampleURLs...)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no Turtle files found")
	}

	file, err := os.CreateTemp("", "epos-import-*.ttl")
	if err != nil {
		return nil, fmt.Errorf("create temporary import file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			display.Warn("close temporary import file: %v", err)
		}
		if err := os.Remove(file.Name()); err != nil {
			display.Warn("remove temporary import file: %v", err)
		}
	}()

	for _, path := range paths {
		if err := appendTurtle(file, path); err != nil {
			return nil, err
		}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek import file: %w", err)
	}

	token := strings.TrimSpace(opts.BearerToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("EPOS_BEARER_TOKEN"))
	}
	if token == "" {
		if !env.Components.AAIService.Enabled {
			return nil, fmt.Errorf("EPOS_BEARER_TOKEN is required for external AAI")
		}
		token, err = env.ManagedAAIToken()
		if err != nil {
			return nil, err
		}
	}
	urls, err := env.BuildEnvURLs()
	if err != nil {
		return nil, fmt.Errorf("build environment URLs: %w", err)
	}
	endpoint, err := url.JoinPath(urls.APIURL, "imports/turtle")
	if err != nil {
		return nil, fmt.Errorf("build import URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, io.NopCloser(file))
	if err != nil {
		return nil, fmt.Errorf("create import request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/turtle")
	res, err := populateClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send import request; outcome may be unknown, check the environment before retrying: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("import request failed with HTTP status %d", res.StatusCode)
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
		return nil, fmt.Errorf("import succeeded but response could not be read; do not retry without checking the environment: %w", err)
	}
	display.Info("Import counts: groups=%d categories=%d organizations=%d contact_points=%d datasets=%d software_applications=%d software_source_codes=%d", result.Groups, result.Categories, result.Organizations, result.ContactPoints, result.Datasets, result.SoftwareApplications, result.SoftwareSourceCodes)
	for _, warning := range result.Warnings {
		display.Warn("Import %s: subject=%s predicate=%s: %s", warning.Severity, warning.Subject, warning.Predicate, warning.Message)
	}
	for _, path := range paths {
		if err := db.InsertIngestedFile(opts.Name, path); err != nil {
			return nil, fmt.Errorf("import succeeded but recording input failed; do not retry without checking the environment: %w", err)
		}
	}
	display.Done("Imported %d Turtle inputs", len(paths))
	return env, nil
}

func collectTurtleFiles(paths []string) ([]string, error) {
	seen := make(map[string]bool)
	var files []string
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve input path: %w", err)
		}
		abs, err = filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, fmt.Errorf("resolve input path %q: %w", path, err)
		}
		if err := filepath.WalkDir(abs, func(path string, entry fs.DirEntry, walkErr error) error {
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
					files = append(files, resolved)
				}
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("collect Turtle files from %q: %w", path, err)
		}
	}
	slices.Sort(files)
	return files, nil
}

func appendTurtle(dst *os.File, path string) error {
	var src io.ReadCloser
	if strings.HasPrefix(path, "https://") {
		res, err := populateClient.Get(path)
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
	pos, err := dst.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("check import size: %w", err)
	}
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

// ManagedAAIToken requests an access token using the configured managed AAI admin account.
func (env *Env) ManagedAAIToken() (string, error) {
	if !env.Components.AAIService.Enabled {
		return "", fmt.Errorf("managed AAI is not enabled")
	}
	endpoint, err := url.JoinPath(env.AAIAuthRootURL(), "oauth/token")
	if err != nil {
		return "", fmt.Errorf("build AAI token URL: %w", err)
	}
	form := url.Values{"grant_type": {"password"}, "username": {env.Components.AAIService.Email}, "password": {env.Components.AAIService.Password}}
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create AAI token request: %w", err)
	}
	req.SetBasicAuth("local-dev-client", "dev-secret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := *populateClient
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

// Validate checks PopulateOpts and verifies that input paths and environment state are valid.
func (p *PopulateOpts) Validate() error {
	if err := EnsureEnvironmentExists(p.Name); err != nil {
		return fmt.Errorf("no environment with name %q exists: %w", p.Name, err)
	}
	if len(p.TTLDirs) == 0 && !p.PopulateExamples {
		return fmt.Errorf("at least one Turtle path or examples is required")
	}
	for _, item := range p.TTLDirs {
		info, err := os.Stat(item)
		if err != nil {
			return fmt.Errorf("stat path %q: %w", item, err)
		}
		if !info.IsDir() && filepath.Ext(item) != ".ttl" {
			return fmt.Errorf("file %s is not a .ttl file", item)
		}
	}
	return nil
}
