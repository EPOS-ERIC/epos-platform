package tui

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"
)

// showCleanConfirm displays a confirmation dialog for cleaning an environment.
func (a *App) showCleanConfirm(envName string, isDocker bool) {
	if envName == "" || !isDocker {
		return
	}

	message := "This will permanently delete managed database and telemetry data in environment '" + envName + "'.\n\n" + DefaultTheme.DestructiveTag("b") + "This action cannot be undone." + "[-]"

	a.UpdateFooter(CleanConfirmKey)

	a.ShowConfirmation(ConfirmationOptions{
		PageName:           "clean-confirm",
		Title:              " [::b]Clean Environment ",
		Message:            message,
		ConfirmLabel:       "Clean",
		CancelLabel:        "Cancel",
		ConfirmDestructive: true,
		Secondary:          true,
		OnConfirm: func() {
			a.showCleanProgress(envName)
		},
		OnCancel: func() {
			a.ResetToHome(ResetOptions{
				PageNames:    []string{"clean-confirm"},
				RestoreFocus: true,
			})
		},
	})
}

// showCleanProgress displays the cleaning progress with live output.
func (a *App) showCleanProgress(envName string) {
	a.RunBackgroundTask(TaskOptions{
		Operation: "Clean",
		EnvName:   envName,
		IsDocker:  true,
		Task: func() (string, error) {
			env, err := docker.Clean(docker.CleanOpts{Name: envName})
			if err != nil {
				return "", err
			}
			urls, err := env.BuildEnvURLs()
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Environment cleaned successfully! GUI: %s", urls.GUIURL), nil
		},
	})
}
