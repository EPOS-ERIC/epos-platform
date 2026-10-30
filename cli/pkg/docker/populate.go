package docker

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/db"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
)

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

	urls, err := env.BuildEnvURLs()
	if err != nil {
		return nil, fmt.Errorf("build environment URLs: %w", err)
	}

	paths, err := common.PopulateTurtle(opts.TTLDirs, opts.PopulateExamples, opts.BearerToken, env.Components.AAIService.Enabled, env.ManagedAAIToken, urls.APIBaseURL)
	if err != nil {
		return nil, err
	}

	for _, path := range paths {
		if err := db.InsertIngestedFile(opts.Name, path); err != nil {
			return nil, fmt.Errorf("import succeeded but recording input failed; do not retry without checking the environment: %w", err)
		}
	}

	display.Done("Imported %d Turtle inputs", len(paths))

	return env, nil
}

// ManagedAAIToken requests an access token using the configured managed AAI admin account.
func (env *Env) ManagedAAIToken() (string, error) {
	if !env.Components.AAIService.Enabled {
		return "", fmt.Errorf("managed AAI is not enabled")
	}

	return common.RequestManagedToken(env.AAIAuthRootURL(), env.Components.AAIService.Email, env.Components.AAIService.Password)
}

// Validate checks PopulateOpts and verifies that input paths and environment state are valid.
func (p *PopulateOpts) Validate() error {
	if err := EnsureEnvironmentExists(p.Name); err != nil {
		return fmt.Errorf("no environment with name %q exists: %w", p.Name, err)
	}

	if len(p.TTLDirs) == 0 && !p.PopulateExamples {
		return fmt.Errorf("at least one Turtle path or examples is required")
	}

	return common.ValidateTurtlePaths(p.TTLDirs, "stat path")
}
