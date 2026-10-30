package docker

import (
	"fmt"
	"os"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker/config"

	"github.com/spf13/cobra"
)

var force bool

var UpdateCmd = &cobra.Command{
	Use:   "update <env-name>",
	Short: "Update an existing environment.",
	Long:  "Update an existing environment. --force removes containers and volumes.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		var cfg *config.EnvConfig
		var err error
		if configFilePath != "" {
			cfg, err = config.LoadConfig(configFilePath)
			if err != nil {
				display.Error("Failed to load config: %v", err)
				os.Exit(1)
			}
		}
		if force {
			display.Warn("Force update permanently removes containers and volumes for environment '%s'. This cannot be undone.", name)
			confirmed, err := common.Confirm("Continue with update? (y/n):")
			if err != nil {
				display.Error("Failed to read confirmation: %v", err)
				os.Exit(1)
			}
			if !confirmed {
				display.Info("Update cancelled.")
				return
			}
		}

		env, err := docker.Update(docker.UpdateOpts{
			PullImages: pullImages,
			Force:      force,
			OldEnvName: name,
			NewConfig:  cfg,
		})
		if err != nil {
			display.Error("%v", err)
			os.Exit(1)
		}

		urls, err := env.BuildEnvURLs()
		if err != nil {
			display.Error("failed to build environment URLs: %v", err)
			os.Exit(1)
		}

		display.URLs(urls.GUIURL, urls.APIDocsURL, fmt.Sprintf("epos-cli docker update %s", name), urls.BackofficeURL, urls.GrafanaURL)
	},
}

func init() {
	UpdateCmd.Flags().BoolVarP(&force, "force", "f", false, "Recreate containers and volumes")
	UpdateCmd.Flags().BoolVarP(&pullImages, "update-images", "u", false, "Pull Docker images before starting")
	UpdateCmd.Flags().StringVar(&configFilePath, "config", "", "Path to YAML configuration file")
}
