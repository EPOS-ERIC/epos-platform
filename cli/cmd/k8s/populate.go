package k8s

import (
	"fmt"
	"os"

	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s"
	"github.com/spf13/cobra"
)

var PopulateCmd = &cobra.Command{
	Use:               "populate <env-name> [ttl-paths...]",
	Short:             "Load TTL data into an environment.",
	Long:              "Load TTL files or bundled example data into an environment. Set EPOS_BEARER_TOKEN for external AAI.",
	ValidArgsFunction: validArgsFunction,
	Args: func(cmd *cobra.Command, args []string) error {
		if populateExamples {
			if len(args) < 1 {
				display.Error("requires environment name when using --example")
				return fmt.Errorf("requires environment name when using --example")
			}
			return nil
		}
		if len(args) < 2 {
			display.Error("requires environment name and at least one TTL path (or use --example flag)")
			return fmt.Errorf("requires environment name and at least one TTL path (or use --example flag)")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ttlPaths := args[1:]

		env, err := k8s.Populate(k8s.PopulateOpts{
			Context:          context,
			TTLDirs:          ttlPaths,
			Name:             name,
			PopulateExamples: populateExamples,
		})
		if err != nil {
			display.Error("%v", err)
			os.Exit(1)
		}

		URLs, err := env.BuildEnvURLs()
		if err != nil {
			display.Error("Failed to build environment URLs: %v", err)
			os.Exit(1)
		}

		display.URLs(URLs.GUIURL, URLs.APIDocsURL, fmt.Sprintf("epos-cli k8s populate %s", name), URLs.BackofficeURL, URLs.GrafanaURL)
	},
}

func init() {
	addContextFlag(PopulateCmd)
	PopulateCmd.Flags().BoolVar(&populateExamples, "example", false, "Load bundled example data")
}
