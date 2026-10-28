package k8s

import (
	"fmt"
	"os"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s"

	"github.com/spf13/cobra"
)

var force bool

var UpdateCmd = &cobra.Command{
	Use:               "update <env-name>",
	Short:             "Update an existing environment.",
	Long:              "Update an existing environment. Changes to managed database initialization settings require confirmation. Use --force to delete and recreate the entire namespace, including unrelated resources and persistent volume claims. AAI data may be lost; backing volumes follow the storage reclaim policy.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: validArgsFunction,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if force {
			display.Warn("Force update deletes the entire namespace, including unrelated resources and persistent volume claims. AAI data may be lost; backing volumes follow the storage reclaim policy.")
		}

		cfg, err := loadConfigIfProvided(configFilePath)
		if err != nil {
			display.Error("%v", err)
			os.Exit(1)
		}
		if !force && cfg != nil {
			oldEnv, err := k8s.GetEnv(name, context)
			if err != nil {
				display.Error("Failed to load environment: %v", err)
				os.Exit(1)
			}
			if k8s.ManagedDatabaseSettingsChanged(&oldEnv.Config, cfg) {
				display.Warn("Managed database username, database name, or password changed. PostgreSQL does not apply these changes to existing data; services may fail to connect.")
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
		}

		env, err := k8s.Update(k8s.UpdateOpts{
			Force:      force,
			OldEnvName: name,
			NewConfig:  cfg,
			Context:    context,
			Timeout:    timeout,
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

		display.URLs(URLs.GUIURL, URLs.APIURL, fmt.Sprintf("epos-cli k8s update %s", name), URLs.BackofficeURL, nil)
	},
}

func init() {
	addContextFlag(UpdateCmd)
	UpdateCmd.Flags().BoolVarP(&force, "force", "f", false, "Reinstall from scratch by deleting the namespace first")
	UpdateCmd.Flags().StringVar(&configFilePath, "config", "", "Path to YAML configuration file")
	UpdateCmd.Flags().DurationVar(&timeout, "timeout", 0, "Operation timeout (default: 5m)")
}
