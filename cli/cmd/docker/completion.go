package docker

import (
	"github.com/EPOS-ERIC/epos-platform/cli/cmd/internal/completion"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"
	"github.com/spf13/cobra"
)

func validArgsFunction(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completion.SharedValidArgs(cmd, args, toComplete, func() ([]string, error) {
		envs, err := docker.List()
		if err != nil {
			return nil, err
		}

		names := make([]string, len(envs))
		for i, d := range envs {
			names[i] = d.Name
		}

		return names, nil
	})
}
