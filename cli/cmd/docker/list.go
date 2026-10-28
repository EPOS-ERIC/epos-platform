package docker

import (
	"os"

	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"

	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List EPOS environments.",
	Long:  "List EPOS environments. Lists local Docker Compose environments and shows their GUI, API, backoffice, and Grafana URLs.",
	Run: func(cmd *cobra.Command, args []string) {
		envs, err := docker.List()
		if err != nil {
			display.Error("Failed to list Docker environments: %v", err)
			os.Exit(1)
		}

		rows := make([][]any, len(envs))
		for i, dockerEnv := range envs {
			urls, err := dockerEnv.BuildEnvURLs()
			if err != nil {
				display.Error("Failed to build environment URLs for %s: %v", dockerEnv.Name, err)
				os.Exit(1)
			}

			apiURL := dockerEnv.GatewayPublicURL() + "/"

			var backofficeURL string
			if urls.BackofficeURL != nil {
				backofficeURL = *urls.BackofficeURL
			}
			var grafanaURL string
			if urls.GrafanaURL != nil {
				grafanaURL = *urls.GrafanaURL
			}

			rows[i] = []any{dockerEnv.Name, urls.GUIURL, apiURL, backofficeURL, grafanaURL}
		}

		headers := []string{"Name", "GUI URL", "API URL", "Backoffice URL", "Grafana URL"}
		display.InfraList(rows, headers, "Installed Docker environments")
	},
}
