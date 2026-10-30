package tui

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"
	dockerconfig "github.com/EPOS-ERIC/epos-platform/cli/pkg/docker/config"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s"
	k8sconfig "github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s/config"
)

// updateFormData holds the form field values.
type updateFormData struct {
	name          string
	k8sContext    string
	timeout       time.Duration // K8s only
	pullImages    bool          // Docker only
	force         bool          // Docker: reset DB; K8s: reset namespace
	configSession *configEditSession
}

func (u *updateFormData) cleanupConfigSession() {
	if u.configSession != nil {
		_ = u.configSession.Cleanup()
		u.configSession = nil
	}
}

// showUpdateForm displays the environment update form for Docker or K8s.
func (a *App) showUpdateForm() {
	envName, isDocker, k8sContext := a.envList.GetSelected()
	if envName == "" {
		return
	}

	a.PushFocus()
	title := "Update Docker Environment"
	if !isDocker {
		title = "Update K8s Environment"
	}
	a.UpdateFooter(UpdateFormKey)

	data := &updateFormData{
		name:       envName,
		k8sContext: k8sContext,
		timeout:    defaultK8sTimeout(),
	}

	fields := []FormField{}

	if isDocker {
		fields = append(fields,
			FormField{Type: "checkbox", Label: "Pull Images", CheckboxChangedFunc: func(checked bool) { data.pullImages = checked }},
			FormField{Type: "checkbox", Label: "Force - deletes volumes", CheckboxChangedFunc: func(checked bool) { data.force = checked }},
		)
	} else {
		fields = append(fields,
			FormField{Type: "checkbox", Label: "Force - deletes namespace and data", CheckboxChangedFunc: func(checked bool) { data.force = checked }},
			FormField{Type: "dropdown", Label: "Timeout", Value: defaultK8sTimeoutOption(), Options: k8sTimeoutOptions(), SelectedFunc: func(option string, index int) {
				data.timeout, _ = time.ParseDuration(option)
			}},
		)
	}

	buttons := []FormButton{
		{Label: "Update", SelectedFunc: func() { a.handleUpdate(data, isDocker) }},
		{Label: "Cancel", SelectedFunc: func() {
			data.cleanupConfigSession()
			a.ResetToHome(ResetOptions{
				PageNames:    []string{"update"},
				RefreshFiles: true,
				RestoreFocus: true,
			})
		}},
	}

	height := 12
	if !isDocker {
		height = 14
	}

	opts := ModalFormOptions{
		PageName: "update",
		Title:    title,
		Fields:   fields,
		Buttons:  buttons,
		Width:    64,
		Height:   height,
		BottomButton: &FormButton{
			Label:        "Edit Config",
			SelectedFunc: func() { a.editUpdateConfig(data, isDocker) },
		},
		OnCancel: func() {
			data.cleanupConfigSession()
			a.ResetToHome(ResetOptions{
				PageNames:    []string{"update"},
				RefreshFiles: true,
				RestoreFocus: true,
			})
		},
	}

	a.ShowModalForm(opts)
}

// handleUpdate validates the form and starts update.
func (a *App) handleUpdate(data *updateFormData, isDocker bool) {
	var timeout time.Duration
	if !isDocker {
		timeout = data.timeout
	}

	dockerCfg, k8sCfg, err := a.buildUpdateConfig(data, isDocker)
	if err != nil {
		a.ShowError(err.Error())
		return
	}

	start := func() {
		data.cleanupConfigSession()
		a.showUpdateProgress(data, isDocker, dockerCfg, k8sCfg, timeout)
	}
	if isDocker && data.force {
		a.ShowConfirmation(ConfirmationOptions{
			PageName:           "update-force-warning",
			Title:              " Force update ",
			Message:            "This permanently removes containers and volumes. This cannot be undone.",
			ConfirmLabel:       "Update",
			CancelLabel:        "Cancel",
			ConfirmDestructive: true,
			OnConfirm:          start,
			OnCancel: func() {
				a.pages.RemovePage("update-force-warning")
				a.PopFocus()
			},
		})
		return
	}
	if !isDocker && !data.force && k8sCfg != nil {
		oldEnv, err := k8s.GetEnv(data.name, data.k8sContext)
		if err != nil {
			a.ShowError(fmt.Sprintf("Failed to load environment: %v", err))
			return
		}
		if k8s.ManagedDatabaseSettingsChanged(&oldEnv.Config, k8sCfg) {
			a.ShowConfirmation(ConfirmationOptions{
				PageName:     "update-db-warning",
				Title:        " Database settings changed ",
				Message:      "PostgreSQL does not apply changed username, database name, or password to existing data. Services may fail to connect.",
				ConfirmLabel: "Update",
				CancelLabel:  "Cancel",
				OnConfirm:    start,
				OnCancel: func() {
					a.pages.RemovePage("update-db-warning")
					a.PopFocus()
				},
			})
			return
		}
	}
	start()
}

// showUpdateProgress displays the update progress with live output.
func (a *App) showUpdateProgress(data *updateFormData, isDocker bool, dockerCfg *dockerconfig.EnvConfig, k8sCfg *k8sconfig.Config, timeout time.Duration) {
	a.RunBackgroundTask(TaskOptions{
		Operation: "Update",
		EnvName:   data.name,
		IsDocker:  isDocker,
		Task: func() (string, error) {
			var err error
			if isDocker {
				_, err = docker.Update(docker.UpdateOpts{
					PullImages: data.pullImages,
					Force:      data.force,
					OldEnvName: data.name,
					NewConfig:  dockerCfg,
				})
			} else {
				_, err = k8s.Update(k8s.UpdateOpts{
					OldEnvName: data.name,
					Context:    data.k8sContext,
					Timeout:    timeout,
					Force:      data.force,
					NewConfig:  k8sCfg,
				})
			}
			return "", err
		},
	})
}

func (a *App) editUpdateConfig(data *updateFormData, isDocker bool) {
	if data.configSession == nil {
		fileName := "docker-config.yaml"
		if !isDocker {
			fileName = "k8s-config.yaml"
		}

		session, err := newConfigEditSession(fileName)
		if err != nil {
			a.ShowError(err.Error())
			return
		}

		if isDocker {
			seed, err := a.loadDockerUpdateSeed(data.name)
			if err != nil {
				loadErr := err
				seed, err = dockerconfig.GetDefaultConfig()
				if err != nil {
					_ = session.Cleanup()
					a.ShowError(err.Error())
					return
				}
				seed.Name = data.name
				a.FlashMessage(fmt.Sprintf("Using default Docker config seed: %v", loadErr), 3*time.Second)
			}

			if err := seed.Save(session.FilePath()); err != nil {
				_ = session.Cleanup()
				a.ShowError(err.Error())
				return
			}
		} else {
			seed, err := a.loadK8sUpdateSeed(data.name, data.k8sContext)
			if err != nil {
				seed = k8sconfig.GetDefaultConfig()
				seed.Name = data.name
				a.FlashMessage(fmt.Sprintf("Using default K8s config seed: %v", err), 3*time.Second)
			}

			if err := seed.Save(session.FilePath()); err != nil {
				_ = session.Cleanup()
				a.ShowError(err.Error())
				return
			}
		}

		data.configSession = session
	}

	if err := a.openConfigEditor(data.configSession.FilePath()); err != nil {
		a.ShowError(err.Error())
		return
	}

	a.FlashMessage("Config opened. Save changes before updating.", 2*time.Second)
}

func (a *App) buildUpdateConfig(data *updateFormData, isDocker bool) (*dockerconfig.EnvConfig, *k8sconfig.Config, error) {
	if data.configSession == nil {
		return nil, nil, nil
	}

	if isDocker {
		cfg, err := dockerconfig.LoadConfig(data.configSession.FilePath())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load docker config from %s: %w", data.configSession.FilePath(), err)
		}

		if cfg.Name == "" {
			cfg.Name = data.name
		}

		if err := cfg.Validate(); err != nil {
			return nil, nil, fmt.Errorf("invalid docker config: %w", err)
		}

		return cfg, nil, nil
	}

	cfg, err := k8sconfig.LoadConfig(data.configSession.FilePath())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load k8s config from %s: %w", data.configSession.FilePath(), err)
	}

	if cfg.Name == "" {
		cfg.Name = data.name
	}

	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid k8s config: %w", err)
	}

	return nil, cfg, nil
}

func (a *App) loadDockerUpdateSeed(envName string) (*dockerconfig.EnvConfig, error) {
	env, err := docker.GetEnv(envName)
	if err != nil {
		return nil, fmt.Errorf("failed to load docker environment %s: %w", envName, err)
	}

	cfg := env.EnvConfig

	return &cfg, nil
}

func (a *App) loadK8sUpdateSeed(envName, context string) (*k8sconfig.Config, error) {
	env, err := k8s.GetEnv(envName, context)
	if err != nil {
		return nil, fmt.Errorf("failed to load k8s environment %s in context %s: %w", envName, context, err)
	}

	cfg := env.Config

	return &cfg, nil
}
