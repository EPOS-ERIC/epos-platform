package docker

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker/config"
)

// ExportOpts defines inputs for Export.
type ExportOpts struct {
	// Required. Path to export the default environment config
	Path string
}

// Export writes the default Docker configuration file to the requested path.
func Export(opts ExportOpts) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid export parameters: %w", err)
	}

	cfg, err := config.GetDefaultConfig()
	if err != nil {
		return fmt.Errorf("create default config: %w", err)
	}
	data, err := cfg.Bytes()
	if err != nil {
		return fmt.Errorf("serialize default config: %w", err)
	}

	display.Debug("loaded default docker config bytes")

	path, err := common.Export(opts.Path, "docker-config.yaml", data)
	if err != nil {
		return fmt.Errorf("failed to export config: %w", err)
	}

	display.Debug("exported docker config path: %s", path)
	display.Done("Exported config: %s", path)
	display.Info("You can now use the config to deploy the environment with 'epos-cli docker deploy --config %s'", path)

	return nil
}

// Validate checks ExportOpts for required values.
func (d *ExportOpts) Validate() error {
	display.Debug("path: %s", d.Path)

	if d.Path == "" {
		return fmt.Errorf("path is required")
	}

	return nil
}
