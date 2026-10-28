package docker

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/cli/command"
	"github.com/EPOS-ERIC/epos-platform/cli/db"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
)

// CleanOpts defines inputs for Clean.
type CleanOpts struct {
	// Required. name of the environment
	Name string
}

// Clean removes runtime data from an existing Docker environment and restarts required services.
func Clean(opts CleanOpts) (*Env, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid clean parameters: %w", err)
	}

	display.Step("Cleaning data in environment: %s", opts.Name)

	env, err := GetEnv(opts.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to load docker environment %s: %w", opts.Name, err)
	}

	display.Step("Stopping managed containers")
	if err := downStack(&env.EnvConfig, false); err != nil {
		display.Warn("Clean failed; managed data may have been partially deleted. Attempting to restart services")
		if restartErr := deployStack(false, &env.EnvConfig); restartErr != nil {
			return nil, errors.Join(fmt.Errorf("remove managed volumes: %w", err), fmt.Errorf("restart services: %w", restartErr))
		}
		return nil, fmt.Errorf("remove managed volumes: %w", err)
	}

	var cleanErr error
	for _, volume := range []string{"metadata-db", "retriever-db", "sharing-db", "otel-lgtm"} {
		names, err := command.RunCommand(exec.Command("docker", "volume", "ls", "-q", "--filter", "label=com.docker.compose.project="+opts.Name, "--filter", "label=com.docker.compose.volume="+volume), true)
		if err != nil {
			cleanErr = fmt.Errorf("find %s volume: %w", volume, err)
			break
		}
		if names = strings.TrimSpace(names); names != "" {
			args := append([]string{"volume", "rm"}, strings.Fields(names)...)
			if _, err := command.RunCommand(exec.Command("docker", args...), false); err != nil {
				cleanErr = fmt.Errorf("remove %s volume: %w", volume, err)
				break
			}
		}
		if volume == "metadata-db" && env.Components.MetadataDatabase.Enabled {
			if err := db.DeleteIngestedFilesByEnvironment(opts.Name); err != nil {
				cleanErr = fmt.Errorf("clear ingested file tracking: %w", err)
				break
			}
		}
	}
	if err := deployStack(false, &env.EnvConfig); err != nil {
		return nil, errors.Join(cleanErr, fmt.Errorf("restart environment after removing volumes: %w", err))
	}
	if cleanErr != nil {
		return nil, fmt.Errorf("clean incomplete; some managed data may have been deleted: %w", cleanErr)
	}

	display.Done("Cleaned environment: %s", opts.Name)

	return env, nil
}

// Validate checks CleanOpts and ensures the target environment exists.
func (c *CleanOpts) Validate() error {
	display.Debug("name: %s", c.Name)

	if err := EnsureEnvironmentExists(c.Name); err != nil {
		return fmt.Errorf("no environment with the name '%s' exists: %w", c.Name, err)
	}

	return nil
}
