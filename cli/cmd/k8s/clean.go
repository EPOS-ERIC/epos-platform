package k8s

import (
	"fmt"
	"os"

	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s"

	"github.com/spf13/cobra"
)

var CleanCmd = &cobra.Command{
	Use:               "clean <env-name>",
	Short:             "Clean is not implemented for Kubernetes environments.",
	Long:              "Kubernetes clean is not implemented. This command returns an error without changing the environment.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: validArgsFunction,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		env, err := k8s.Clean(k8s.CleanOpts{
			Name:    name,
			Context: context,
			Timeout: timeout,
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

		display.URLs(URLs.GUIURL, URLs.APIURL, fmt.Sprintf("epos-cli k8s clean %s", name), URLs.BackofficeURL, nil)
	},
}

func init() {
	addContextFlag(CleanCmd)
	CleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip the confirmation prompt")
	CleanCmd.Flags().DurationVar(&timeout, "timeout", 0, "Operation timeout (default: 5m)")
}
