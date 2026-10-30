// Package k8s contains the internal functions used by the k8s cmd to manage the environments
package k8s

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s/config"
	"github.com/EPOS-ERIC/epos-platform/cli/validate"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// UpdateOpts defines inputs for Update.
type UpdateOpts struct {
	// Required. name of the environment
	OldEnvName string
	// Optional. Kubernetes context to use; defaults to the current kubectl context when unset.
	Context string
	// Optional. Timeout for Helm operations; defaults when unset.
	Timeout time.Duration
	// Optional. do an uninstall then an install (to wipe volumes and stuff) maybe we should rename the flag?
	Force bool
	// New configuration to apply. If nil, preserves existing config
	NewConfig *config.Config
}

// Update updates an existing K8s environment in place or by full reinstall when Force is enabled.
func Update(opts UpdateOpts) (*Env, error) {
	display.Debug("oldEnvName: %s", opts.OldEnvName)
	display.Debug("context: %s", opts.Context)
	display.Debug("force: %v", opts.Force)

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters for update command: %w", err)
	}

	if opts.NewConfig == nil {
		display.Debug("new config not provided, using current environment config")

		oldEnv, err := GetEnv(opts.OldEnvName, opts.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to get environment: %w", err)
		}

		opts.NewConfig = &oldEnv.Config
	}

	display.Step("Updating environment: %s", opts.OldEnvName)

	display.Step("Updating stack")

	// if force do uninstall + install again
	if opts.Force {
		display.Info("Force flag enabled: reinstalling environment")

		if err := Delete(DeleteOpts{
			Name:    []string{opts.OldEnvName},
			Context: opts.Context,
			Timeout: opts.Timeout,
		}); err != nil {
			return nil, fmt.Errorf("failed to delete environment: %w", err)
		}
		env, err := Deploy(DeployOpts{
			Context: opts.Context,
			Timeout: opts.Timeout,
			Config:  opts.NewConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to deploy new environment: %w", err)
		}

		display.Done("Updated environment: %s", opts.OldEnvName)

		return env, nil
	}

	display.Debug("building chart")

	chart, err := config.GetChart()
	if err != nil {
		return nil, fmt.Errorf("failed to load helm chart: %w", err)
	}

	display.Debug("building values")

	values, err := opts.NewConfig.AsValues()
	if err != nil {
		return nil, fmt.Errorf("failed to build helm values from config: %w", err)
	}

	settings := cli.New()
	settings.KubeContext = opts.Context
	settings.SetNamespace(opts.NewConfig.Name)

	actionConfig := &action.Configuration{}
	if err := actionConfig.Init(
		settings.RESTClientGetter(),
		opts.NewConfig.Name,
		"secret",
		func(format string, v ...any) { display.Info(format, v...) },
	); err != nil {
		return nil, fmt.Errorf("failed to init helm action config: %w", err)
	}

	client := action.NewUpgrade(actionConfig)
	client.Namespace = opts.NewConfig.Name
	client.Wait = true
	client.WaitForJobs = true
	client.Atomic = true
	client.Timeout = opts.Timeout

	display.Debug("running helm upgrade")

	rel, err := client.Run(opts.NewConfig.Name, chart, values.AsMap())
	if err != nil {
		return nil, fmt.Errorf("failed to install helm chart: %w", err)
	}

	display.Debug("upgraded release: %s (status: %s)", rel.Name, rel.Info.Status)

	newEnv, err := EnvFromRelease(rel, opts.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to convert upgraded release to environment: %w", err)
	}

	display.Done("Updated environment: %s", opts.OldEnvName)

	return newEnv, nil
}

// Validate checks UpdateOpts and validates update semantics against existing environment state.
func (u *UpdateOpts) Validate() error {
	display.Debug("oldEnvName: %s", u.OldEnvName)
	display.Debug("context: %s", u.Context)
	display.Debug("force: %v", u.Force)

	if u.OldEnvName == "" {
		return fmt.Errorf("old environment name is required")
	}

	if err := validate.Name(u.OldEnvName); err != nil {
		return fmt.Errorf("'%s' is an invalid name for an environment: %w", u.OldEnvName, err)
	}

	resolvedTimeout, err := resolveCommandTimeout(u.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	u.Timeout = resolvedTimeout

	if u.NewConfig != nil {
		if u.NewConfig.Name == "" {
			u.NewConfig.Name = u.OldEnvName

			display.Debug("new config name not set, using old name: %s", u.OldEnvName)
		}

		if u.NewConfig.Name != u.OldEnvName {
			return fmt.Errorf("config name %q must match environment name %q", u.NewConfig.Name, u.OldEnvName)
		}

		if err := u.NewConfig.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

	}

	usingCurrentContext := u.Context == ""
	u.Context, err = resolveContext(u.Context)
	if err != nil {
		return err
	}
	if usingCurrentContext {
		display.Debug("using current kubectl context: %s", u.Context)
	}

	if err := EnsureEnvironmentExists(u.OldEnvName, u.Context); err != nil {
		return fmt.Errorf("no environment with the name '%s' exists: %w", u.OldEnvName, err)
	}

	return nil
}

// ManagedDatabaseSettingsChanged reports settings PostgreSQL only applies on initial startup.
func ManagedDatabaseSettingsChanged(old, new *config.Config) bool {
	if old == nil || new == nil {
		return false
	}
	for _, pair := range [][2]config.MetadataDatabase{
		{old.Components.MetadataDatabase, new.Components.MetadataDatabase},
		{old.Components.DataRetrieverDatabase, new.Components.DataRetrieverDatabase},
		{old.Components.SharingDatabase, new.Components.SharingDatabase},
	} {
		oldDB, newDB := pair[0], pair[1]
		if oldDB.Enabled && newDB.Enabled && (oldDB.User != newDB.User || oldDB.DBName != newDB.DBName || oldDB.Password != newDB.Password) {
			return true
		}
	}
	return false
}
