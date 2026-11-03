package tui

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s"
	"github.com/rivo/tview"
)

// showDeleteConfirm displays a confirmation dialog for deleting a Docker or K8s environment.
func (a *App) showDeleteConfirm(envName string, isDocker bool, k8sContext string) {
	if envName == "" {
		return
	}

	message := fmt.Sprintf("Environment: %s\n\nThis will permanently remove all containers, volumes, and associated resources.\n\n", tview.Escape(envName))
	if !isDocker {
		message = fmt.Sprintf("Environment: %s\n", tview.Escape(envName))
		if k8sContext != "" {
			message += fmt.Sprintf("Context: %s\n", tview.Escape(k8sContext))
		}
		message += "\nThis will permanently remove the namespace and all associated K8s resources.\n\n"
	}
	message += DefaultTheme.DestructiveTag("b") + "This action cannot be undone." + "[-]"

	a.UpdateFooter(DeleteConfirmKey)

	a.ShowConfirmation(ConfirmationOptions{
		PageName:           "delete-confirm",
		Title:              " [::b]Delete Environment ",
		Message:            message,
		ConfirmLabel:       "Delete",
		CancelLabel:        "Cancel",
		Destructive:        true,
		ConfirmDestructive: true,
		OnConfirm: func() {
			a.showDeleteProgress(envName, isDocker, k8sContext)
		},
		OnCancel: func() {
			a.ResetToHome(ResetOptions{
				PageNames:    []string{"delete-confirm"},
				RestoreFocus: true,
			})
		},
	})
}

// showDeleteProgress displays the deletion progress with live output.
func (a *App) showDeleteProgress(envName string, isDocker bool, context string) {
	a.RunBackgroundTask(TaskOptions{
		Operation: "Delete",
		EnvName:   envName,
		IsDocker:  isDocker,
		Task: func() (string, error) {
			if isDocker {
				return "", docker.Delete(docker.DeleteOpts{
					Name: []string{envName},
				})
			}
			return "", k8s.Delete(k8s.DeleteOpts{
				Name:    []string{envName},
				Context: context,
			})
		},
	})
}
