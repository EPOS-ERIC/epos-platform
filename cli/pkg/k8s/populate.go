package k8s

import (
	"fmt"
	"net/url"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
)

// PopulateOpts defines inputs for Populate.
type PopulateOpts struct {
	Name             string
	Context          string
	TTLDirs          []string
	PopulateExamples bool
	BearerToken      string
}

// Populate imports Turtle inputs into an existing Kubernetes environment.
func Populate(opts PopulateOpts) (*Env, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters for populate command: %w", err)
	}

	env, err := GetEnv(opts.Name, opts.Context)
	if err != nil {
		return nil, fmt.Errorf("error getting environment: %w", err)
	}

	urls, err := env.BuildEnvURLs()
	if err != nil {
		return nil, fmt.Errorf("build environment URLs: %w", err)
	}

	paths, err := common.PopulateTurtle(opts.TTLDirs, opts.PopulateExamples, opts.BearerToken, env.Components.AAIService.Enabled, env.ManagedAAIToken, urls.APIBaseURL)
	if err != nil {
		return nil, err
	}

	display.Done("Imported %d Turtle inputs", len(paths))

	return env, nil
}

// ManagedAAIToken requests a token from the managed AAI service.
func (env *Env) ManagedAAIToken() (string, error) {
	if !env.Components.AAIService.Enabled {
		return "", fmt.Errorf("managed AAI is not enabled")
	}

	base := (&url.URL{Scheme: env.Protocol, Host: env.Domain}).String()

	var err error
	if env.URLPrefixNamespace {
		base, err = url.JoinPath(base, env.Name)
		if err != nil {
			return "", fmt.Errorf("build AAI URL: %w", err)
		}
	}

	base, err = url.JoinPath(base, "aai")
	if err != nil {
		return "", fmt.Errorf("build AAI URL: %w", err)
	}

	admin := env.Components.AAIService

	return common.RequestManagedToken(base, admin.Email, admin.Password)
}

// Validate checks PopulateOpts and verifies paths, context, and environment state.
func (p *PopulateOpts) Validate() error {
	if len(p.TTLDirs) == 0 && !p.PopulateExamples {
		return fmt.Errorf("at least one Turtle path or examples is required")
	}

	context, err := resolveContext(p.Context)
	if err != nil {
		return err
	}
	p.Context = context

	if err := EnsureEnvironmentExists(p.Name, p.Context); err != nil {
		return fmt.Errorf("error validating environment name, no environment with '%s' exists: %w", p.Name, err)
	}

	return common.ValidateTurtlePaths(p.TTLDirs, "error stating path")
}
