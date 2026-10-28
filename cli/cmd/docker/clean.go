package docker

import (
	"fmt"
	"os"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"

	"github.com/spf13/cobra"
)

var CleanCmd = &cobra.Command{
	Use:               "clean <env-name>",
	Short:             "Reset an environment's data.",
	Long:              "Reset managed database and telemetry volumes, then restart the environment. AAI data and external resources are preserved. Prompts for confirmation unless --force is set.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: validArgsFunction,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if !cleanForce {
			display.Warn("This will permanently delete all data in environment '%s'. This action cannot be undone.", name)
			confirmed, err := common.Confirm("Are you sure you want to continue? (y/n):")
			if err != nil {
				display.Error("Failed to read confirmation: %v", err)
				os.Exit(1)
			}
			if !confirmed {
				display.Info("Clean operation cancelled.")
				return
			}
		}

		env, err := docker.Clean(docker.CleanOpts{
			Name: name,
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

		display.URLs(urls.GUIURL, env.GatewayPublicURL()+"/", fmt.Sprintf("epos-cli docker clean %s", name), urls.BackofficeURL, urls.GrafanaURL)
	},
}

func init() {
	CleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip the confirmation prompt")
}
