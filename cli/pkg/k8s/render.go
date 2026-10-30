package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s/config"
	"github.com/EPOS-ERIC/epos-platform/cli/validate"
)

// RenderOpts defines inputs for Render.
type RenderOpts struct {
	// Optional. Name to give to the environment. If not set it will use the name set in the passed config. If set and also set in the config, this has precedence
	Name string
	// Optional. Custom config for the environment.
	Config *config.Config
	// Optional. Path to write rendered Kubernetes manifest files.
	OutputPath string
}

// Render materializes rendered K8s manifests from configuration and returns created file paths.
func Render(opts RenderOpts) ([]string, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid render parameters: %w", err)
	}

	display.Debug("rendering k8s templates")
	files, err := opts.Config.Render()
	if err != nil {
		return nil, fmt.Errorf("failed to render templates: %w", err)
	}
	display.Debug("rendered k8s templates: %d", len(files))

	opts.OutputPath, err = common.RenderOutputPath(opts.OutputPath)
	if err != nil {
		return nil, err
	}

	outputPaths := []string{}
	for name, out := range files {
		// don't save empty files
		out = strings.TrimSpace(out)
		if out == "" || out == "---" {
			continue
		}

		filePath := filepath.Join(opts.OutputPath, name)

		display.Debug("writing file: %s", filePath)

		if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
			return nil, fmt.Errorf("failed to create parent directory for %s file: %w", name, err)
		}

		err := common.CreateFileWithContent(filePath, out, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s file: %w", name, err)
		}

		outputPaths = append(outputPaths, filePath)
	}

	display.Done("Rendered K8s config in: %s", opts.OutputPath)

	return outputPaths, nil
}

// Validate checks RenderOpts and applies defaults needed before rendering.
func (r *RenderOpts) Validate() error {
	display.Debug("name: %s", r.Name)
	display.Debug("outputPath: %s", r.OutputPath)

	if r.Config == nil {
		r.Config = config.GetDefaultConfig()

		display.Debug("config not provided, using default config")
	}

	if r.Name == "" {
		r.Name = r.Config.Name
	}

	if r.Name == "" {
		return fmt.Errorf("environment name is required")
	}

	if err := validate.Name(r.Name); err != nil {
		return fmt.Errorf("invalid name for environment: %w", err)
	}

	r.Config.Name = r.Name

	display.Debug("validated render config name: %s", r.Config.Name)

	return nil
}
