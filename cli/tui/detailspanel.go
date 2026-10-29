package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/db"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

// DetailRow represents a row in the details grid.
type DetailRow struct {
	Label            string
	Value            string
	IncludeOpen      bool
	OnCopy           func()
	OnOpen           func()
	ExtraActionLabel string
	ExtraAction      func()
}

// DetailsPanel manages the right-side information and action panel.
type DetailsPanel struct {
	app                   *App
	details               *tview.Flex
	detailsGrid           *tview.Grid
	detailsButtons        []*tview.Button
	nameDirGrid           *tview.Grid
	nameDirButtons        []*tview.Button
	buttonsFlex           *tview.Flex
	deleteButton          *tview.Button
	cleanButton           *tview.Button
	updateButton          *tview.Button
	renderButton          *tview.Button
	populateButton        *tview.Button
	detailsList           *tview.List
	detailsListEmpty      *tview.TextView
	detailsListFlex       *tview.Flex
	detailsEmpty          *tview.TextView
	currentDetailsName    string
	currentDetailsType    string
	currentDetailsContext string
	currentDetailsRows    []DetailRow
	configViewSession     *configEditSession
	detailsShown          bool
}

// NewDetailsPanel creates a new DetailsPanel component.
func NewDetailsPanel(app *App) *DetailsPanel {
	dp := &DetailsPanel{app: app}
	dp.buildUI()
	return dp
}

// GetFlex returns the main flex for this component.
func (dp *DetailsPanel) GetFlex() *tview.Flex {
	return dp.details
}

// IsShown returns true if the details panel is currently showing details.
func (dp *DetailsPanel) IsShown() bool {
	return dp.detailsShown
}

// GetCurrentDetailsName returns the current details environment name.
func (dp *DetailsPanel) GetCurrentDetailsName() string {
	return dp.currentDetailsName
}

// GetCurrentDetailsType returns the current details environment type.
func (dp *DetailsPanel) GetCurrentDetailsType() string {
	return dp.currentDetailsType
}

// GetCurrentDetailsContext returns the current details environment context.
func (dp *DetailsPanel) GetCurrentDetailsContext() string {
	return dp.currentDetailsContext
}

func (dp *DetailsPanel) cleanupConfigViewSession() {
	if dp.configViewSession == nil {
		return
	}

	_ = dp.configViewSession.Cleanup()
	dp.configViewSession = nil
}

func (dp *DetailsPanel) currentAppliedConfigYAML() (string, error) {
	switch dp.currentDetailsType {
	case string(DockerKey):
		env, err := docker.GetEnv(dp.currentDetailsName)
		if err != nil {
			return "", fmt.Errorf("failed to load docker environment: %w", err)
		}

		cfgBytes, err := env.Bytes()
		if err != nil {
			return "", fmt.Errorf("failed to marshal docker config: %w", err)
		}

		return string(cfgBytes), nil
	case string(K8sKey):
		env, err := k8s.GetEnv(dp.currentDetailsName, dp.currentDetailsContext)
		if err != nil {
			return "", fmt.Errorf("failed to load k8s environment: %w", err)
		}

		cfgBytes, err := yaml.Marshal(env.Config)
		if err != nil {
			return "", fmt.Errorf("failed to marshal k8s config: %w", err)
		}

		return string(cfgBytes), nil
	default:
		return "", fmt.Errorf("unsupported environment type: %s", dp.currentDetailsType)
	}
}

func (dp *DetailsPanel) copyCurrentAppliedConfig() {
	content, err := dp.currentAppliedConfigYAML()
	if err != nil {
		dp.app.ShowError(fmt.Sprintf("Failed to get applied config: %v", err))
		return
	}

	dp.app.copyToClipboardWithFeedback(content, "Applied config copied to clipboard", "Failed to copy applied config")
}

func (dp *DetailsPanel) openCurrentAppliedConfig() {
	content, err := dp.currentAppliedConfigYAML()
	if err != nil {
		dp.app.ShowError(fmt.Sprintf("Failed to get applied config: %v", err))
		return
	}

	snapshot := "# Applied configuration snapshot (read-only)\n" +
		"# Changes to this file do not update the deployed environment.\n\n" +
		content

	dp.cleanupConfigViewSession()

	fileName := fmt.Sprintf("%s-%s-applied-config.yaml", dp.currentDetailsType, dp.currentDetailsName)
	session, err := newReadOnlyConfigSession(fileName, snapshot)
	if err != nil {
		dp.app.ShowError(fmt.Sprintf("Failed to create config snapshot: %v", err))
		return
	}

	dp.configViewSession = session

	if err := dp.app.openConfigEditor(session.FilePath()); err != nil {
		dp.app.ShowError(err.Error())
	}
}

// buildUI constructs the component layout.
func (dp *DetailsPanel) buildUI() {
	dp.detailsGrid = tview.NewGrid()

	dp.nameDirGrid = tview.NewGrid()
	dp.nameDirGrid.SetBorderPadding(1, 0, 0, 0)

	dp.nameDirButtons = []*tview.Button{}

	dp.deleteButton = NewStyledButton("Delete", func() {
		dp.app.showDeleteConfirm()
	})

	dp.cleanButton = NewStyledButton("Clean", func() {
		dp.app.showCleanConfirm()
	})

	dp.updateButton = NewStyledButton("Update", func() {
		dp.app.showUpdateForm()
	})

	dp.renderButton = NewStyledButton("Render", func() {
		dp.app.showRenderForm()
	})

	dp.populateButton = NewStyledButton("Populate", func() {
		dp.app.showPopulateForm()
	})

	dp.buttonsFlex = tview.NewFlex().SetDirection(tview.FlexColumn)
	dp.updateActionButtons()

	dp.detailsList = NewStyledList()
	dp.detailsList.SetBorder(true)
	dp.detailsList.SetTitle(" [::b]Ingested Files ")
	dp.detailsList.SetTitleColor(DefaultTheme.Secondary)
	dp.detailsList.SetBorderPadding(1, 0, 1, 1)

	dp.detailsListEmpty = NewStyledTextView()
	dp.detailsListEmpty.SetBorder(true)
	dp.detailsListEmpty.SetTitle(" [::b]Ingested Files ")
	dp.detailsListEmpty.SetTitleColor(DefaultTheme.Secondary)
	dp.detailsListEmpty.SetTextAlign(tview.AlignCenter)
	dp.detailsListEmpty.SetText("\n" + DefaultTheme.MutedTag("i") + "No ingested files yet")

	dp.detailsListFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	dp.detailsListFlex.AddItem(dp.detailsList, 0, 1, false)

	dp.detailsEmpty = NewStyledTextView()
	dp.detailsEmpty.SetText(emptyDetailsText())
	dp.detailsEmpty.SetTextAlign(tview.AlignCenter)
	dp.detailsEmpty.SetWordWrap(false)
	dp.detailsEmpty.SetTextColor(DefaultTheme.OnSurface)

	dp.details = tview.NewFlex().SetDirection(tview.FlexRow)
	dp.details.SetBorder(true)
	dp.details.SetBorderColor(DefaultTheme.Surface)
	dp.details.SetTitle(" [::b]Environment Details ")
	dp.details.SetTitleColor(DefaultTheme.Secondary)
	dp.details.SetBorderPadding(1, 0, 1, 1)
	dp.details.AddItem(dp.detailsEmpty, 0, 1, true)
}

func emptyDetailsText() string {
	return strings.Join([]string{
		DefaultTheme.PrimaryTag("") + display.LogoHollow + "[-]",
		DefaultTheme.MutedTag("i") + common.GetVersion(),
		"",
		"",
		"Manage local EPOS environments.",
		"",
		"",
		"",
		"",
		"",
		"",
		DefaultTheme.SecondaryTag("b") + "QUICK START",
		DefaultTheme.MutedTag("") + strings.Repeat("─", 61),
		quickStartRow("↑↓", "Navigate environments", "enter", "Open details"),
		quickStartRow("tab", "Switch Docker / K8s", "n", "Create environment"),
		quickStartRow("?", "View all shortcuts", "q", "Quit"),
	}, "\n")
}

func quickStartRow(leftKey, leftText, rightKey, rightText string) string {
	const (
		keyWidth  = 6
		textWidth = 24
		lastWidth = 19
	)

	return DefaultTheme.PrimaryTag("b") + padRight(leftKey, keyWidth) + "  " +
		DefaultTheme.Tag(DefaultTheme.OnSurface, "") + padRight(leftText, textWidth) + "  " +
		DefaultTheme.PrimaryTag("b") + padRight(rightKey, keyWidth) + "  " +
		DefaultTheme.Tag(DefaultTheme.OnSurface, "") + padRight(rightText, lastWidth)
}

func padRight(value string, width int) string {
	padding := width - utf8.RuneCountInString(value)
	if padding <= 0 {
		return value
	}

	return value + strings.Repeat(" ", padding)
}

func (dp *DetailsPanel) updateActionButtons() {
	dp.buttonsFlex.Clear()
	dp.buttonsFlex.AddItem(tview.NewBox(), 0, 1, false)
	dp.buttonsFlex.AddItem(dp.populateButton, 14, 0, true)
	dp.buttonsFlex.AddItem(tview.NewBox(), 0, 1, false)
	dp.buttonsFlex.AddItem(dp.renderButton, 12, 0, false)
	dp.buttonsFlex.AddItem(tview.NewBox(), 0, 1, false)
	dp.buttonsFlex.AddItem(dp.updateButton, 12, 0, false)
	dp.buttonsFlex.AddItem(tview.NewBox(), 0, 1, false)
	if dp.currentDetailsType == string(DockerKey) {
		dp.buttonsFlex.AddItem(dp.cleanButton, 11, 0, false)
		dp.buttonsFlex.AddItem(tview.NewBox(), 0, 1, false)
	}
	dp.buttonsFlex.AddItem(dp.deleteButton, 12, 0, false)
	dp.buttonsFlex.AddItem(tview.NewBox(), 0, 1, false)
}

// Update fetches and displays environment details in the panel.
func (dp *DetailsPanel) Update(name, envType, context string, focus bool) {
	dp.cleanupConfigViewSession()

	dp.currentDetailsName = name
	dp.currentDetailsType = envType
	dp.currentDetailsContext = context
	dp.updateActionButtons()

	nameDirGridCount := 2
	detailsGridCount := 2

	switch envType {
	case string(DockerKey):
		dp.currentDetailsContext = ""
		if env, err := docker.GetEnv(name); err != nil {
			dp.detailsGrid.Clear()
			dp.detailsButtons = nil
			dp.detailsGrid.SetRows(1)
			dp.detailsGrid.SetColumns(1)
			errorTV := tview.NewTextView().SetText(fmt.Sprintf("Error fetching details for %s: %v", name, err)).SetTextColor(DefaultTheme.Destructive)
			dp.detailsGrid.AddItem(errorTV, 0, 0, 1, 1, 0, 0, false)
		} else {
			urls, err := env.BuildEnvURLs()
			if err != nil {
				dp.app.ShowError(fmt.Sprintf("Error building Docker URLs: %v", err))
				return
			}

			apiURL := urls.APIDocsURL
			var backofficeURL string
			if urls.BackofficeURL != nil {
				backofficeURL = *urls.BackofficeURL
			}
			nameDirRows := []DetailRow{
				{Label: "Name", Value: env.Name, IncludeOpen: false},
				{
					Label:       "Applied Config",
					Value:       "Runtime YAML snapshot",
					IncludeOpen: true,
					OnCopy:      dp.copyCurrentAppliedConfig,
					OnOpen:      dp.openCurrentAppliedConfig,
				},
			}
			nameDirGridCount = len(nameDirRows)
			dp.createGridRows(dp.nameDirGrid, nameDirRows, &dp.nameDirButtons, "Basic Information")

			guiRow := DetailRow{Label: "GUI", Value: urls.GUIURL, IncludeOpen: true}
			backofficeRow := DetailRow{Label: "Backoffice", Value: backofficeURL, IncludeOpen: true}
			if env.Components.AAIService.Enabled {
				copyPassword := func() { dp.copyPassword(true) }
				guiRow.ExtraActionLabel, guiRow.ExtraAction = "Copy password", copyPassword
				backofficeRow.ExtraActionLabel, backofficeRow.ExtraAction = "Copy password", copyPassword
			}
			rows := []DetailRow{guiRow}
			if backofficeURL != "" {
				rows = append(rows, backofficeRow)
			}
			apiRow := DetailRow{Label: "API", Value: apiURL, IncludeOpen: true}
			if env.Components.AAIService.Enabled {
				apiRow.ExtraActionLabel = "Copy token"
				apiRow.ExtraAction = dp.copyToken
			}
			rows = append(rows, apiRow)
			if urls.GrafanaURL != nil {
				rows = append(rows, DetailRow{Label: "Grafana", Value: *urls.GrafanaURL, IncludeOpen: true, ExtraActionLabel: "Copy password", ExtraAction: func() {
					dp.copyPassword(false)
				}})
			}
			detailsGridCount = len(rows)
			dp.currentDetailsRows = rows
			dp.createDetailsRows(rows)
		}
	case string(K8sKey):
		env, err := k8s.GetEnv(name, context)
		if err != nil {
			dp.detailsGrid.Clear()
			dp.detailsButtons = nil
			dp.detailsGrid.SetRows(1)
			dp.detailsGrid.SetColumns(1)
			errorTV := tview.NewTextView().SetText(fmt.Sprintf("Error fetching details for %s: %v", name, err)).SetTextColor(DefaultTheme.Destructive)
			dp.detailsGrid.AddItem(errorTV, 0, 0, 1, 1, 0, 0, false)
		} else {
			urls, err := env.BuildEnvURLs()
			if err != nil {
				dp.detailsGrid.Clear()
				dp.detailsButtons = nil
				dp.detailsGrid.SetRows(1)
				dp.detailsGrid.SetColumns(1)
				errorTV := tview.NewTextView().SetText(fmt.Sprintf("Error building URLs for %s: %v", name, err)).SetTextColor(DefaultTheme.Destructive)
				dp.detailsGrid.AddItem(errorTV, 0, 0, 1, 1, 0, 0, false)
				break
			}

			dp.currentDetailsContext = env.Context

			nameDirRows := []DetailRow{
				{Label: "Name", Value: env.Name, IncludeOpen: false},
				{Label: "Context", Value: env.Context, IncludeOpen: false},
				{
					Label:       "Applied Config",
					Value:       "Runtime YAML snapshot",
					IncludeOpen: true,
					OnCopy:      dp.copyCurrentAppliedConfig,
					OnOpen:      dp.openCurrentAppliedConfig,
				},
			}
			nameDirGridCount = len(nameDirRows)
			dp.createGridRows(dp.nameDirGrid, nameDirRows, &dp.nameDirButtons, "Basic Information")

			guiRow := DetailRow{Label: "GUI", Value: urls.GUIURL, IncludeOpen: true}
			if env.Components.AAIService.Enabled {
				guiRow.ExtraActionLabel = "Copy password"
				guiRow.ExtraAction = func() { dp.copyPassword(true) }
			}
			rows := []DetailRow{guiRow}
			if urls.BackofficeURL != nil {
				backofficeRow := DetailRow{Label: "Backoffice", Value: *urls.BackofficeURL, IncludeOpen: true}
				if env.Components.AAIService.Enabled {
					backofficeRow.ExtraActionLabel = "Copy password"
					backofficeRow.ExtraAction = func() { dp.copyPassword(true) }
				}
				rows = append(rows, backofficeRow)
			}
			apiRow := DetailRow{Label: "API", Value: urls.APIDocsURL, IncludeOpen: true}
			if env.Components.AAIService.Enabled {
				apiRow.ExtraActionLabel = "Copy token"
				apiRow.ExtraAction = dp.copyToken
			}
			rows = append(rows, apiRow)
			if urls.GrafanaURL != nil {
				rows = append(rows, DetailRow{Label: "Grafana", Value: *urls.GrafanaURL, IncludeOpen: true, ExtraActionLabel: "Copy password", ExtraAction: func() { dp.copyPassword(false) }})
			}
			detailsGridCount = len(rows)
			dp.currentDetailsRows = rows
			dp.createDetailsRows(rows)
		}
	}

	// Recalculate heights when optional rows change between environments.
	nameDirGridSize := (nameDirGridCount * 2) + 3 + 1
	detailsGridSize := (detailsGridCount * 2) + 3
	dp.details.Clear()
	dp.details.AddItem(dp.buttonsFlex, 1, 0, true)
	dp.details.AddItem(dp.nameDirGrid, nameDirGridSize, 0, false)
	dp.details.AddItem(dp.detailsGrid, detailsGridSize, 0, false)
	dp.details.AddItem(dp.detailsListFlex, 0, 1, false)
	if !dp.detailsShown {
		dp.detailsShown = true
		updateBoxStyle(dp.details, true)
	}

	dp.RefreshFiles()

	if focus {
		dp.app.tview.SetFocus(dp.details)
		dp.app.UpdateFooter(getDetailsKey(envType))
	}
}

func (dp *DetailsPanel) copyPassword(aai bool) {
	var password string
	if dp.currentDetailsType == string(K8sKey) {
		env, err := k8s.GetEnv(dp.currentDetailsName, dp.currentDetailsContext)
		if err != nil {
			dp.app.ShowError(fmt.Sprintf("Failed to load Kubernetes environment: %v", err))
			return
		}
		password = env.Components.OpenTelemetry.AdminPassword
		if aai {
			password = env.Components.AAIService.Password
		}
	} else {
		env, err := docker.GetEnv(dp.currentDetailsName)
		if err != nil {
			dp.app.ShowError(fmt.Sprintf("Failed to load Docker environment: %v", err))
			return
		}
		password = env.Components.OpenTelemetry.AdminPassword
		if aai {
			password = env.Components.AAIService.Password
		}
	}
	if password == "" {
		dp.app.ShowError("No initial password is configured")
		return
	}
	dp.app.copyToClipboardWithFeedback(password, "Initial password copied to clipboard", "Failed to copy password")
}

func (dp *DetailsPanel) copyToken() {
	name := dp.currentDetailsName
	envType := dp.currentDetailsType
	context := dp.currentDetailsContext
	go func() {
		var token string
		var err error
		if envType == string(K8sKey) {
			var env *k8s.Env
			env, err = k8s.GetEnv(name, context)
			if err == nil {
				token, err = env.ManagedAAIToken()
			}
		} else {
			var env *docker.Env
			env, err = docker.GetEnv(name)
			if err == nil {
				token, err = env.ManagedAAIToken()
			}
		}
		if err == nil {
			dp.app.copyToClipboardWithFeedback(token, "API token copied to clipboard", "Failed to copy API token")
			return
		}
		dp.app.tview.QueueUpdateDraw(func() {
			dp.app.ShowError(fmt.Sprintf("Failed to generate API token: %v", err))
		})
	}()
}

// Clear shows the placeholder text in the details panel.
func (dp *DetailsPanel) Clear() {
	if dp.detailsShown {
		dp.cleanupConfigViewSession()

		dp.details.Clear()
		dp.details.AddItem(dp.detailsEmpty, 0, 1, true)
		dp.detailsShown = false
		updateBoxStyle(dp.details, false)
		dp.nameDirGrid.Clear()
		dp.nameDirButtons = nil
		dp.currentDetailsName = ""
		dp.currentDetailsType = ""
		dp.currentDetailsContext = ""
	}
}

// RefreshFiles refreshes the ingested files list if details are currently shown.
func (dp *DetailsPanel) RefreshFiles() {
	if dp.detailsShown {
		dp.populateIngestedFilesList()
	}
}

// populateIngestedFilesList populates the ingested files list.
func (dp *DetailsPanel) populateIngestedFilesList() {
	dp.detailsList.Clear()
	dp.detailsListFlex.Clear()
	dp.detailsList.SetSelectedFunc(nil)
	dp.detailsList.SetTitle(" [::b]Ingested Files ")
	dp.detailsListEmpty.SetTitle(" [::b]Ingested Files ")

	if dp.currentDetailsType == string(K8sKey) {
		dp.detailsListEmpty.SetText("\n" + DefaultTheme.MutedTag("i") + "Tracking is available only for Docker environments")
		dp.detailsListFlex.AddItem(dp.detailsListEmpty, 0, 1, true)
		dp.syncIngestedFilesFocus()

		return
	}

	dp.detailsListEmpty.SetText("\n" + DefaultTheme.MutedTag("i") + "No ingested files yet")

	if ingestedFiles, err := db.GetIngestedFilesByEnvironment(dp.currentDetailsName); err != nil {
		dp.detailsListEmpty.SetText("\n" + DefaultTheme.DestructiveTag("i") + fmt.Sprintf("Error loading files: %v", err))
		dp.detailsListFlex.AddItem(dp.detailsListEmpty, 0, 1, true)
	} else {
		count := len(ingestedFiles)
		if count > 0 {
			dp.detailsList.SetTitle(fmt.Sprintf(" [::b]Ingested Files (%d) ", count))
			for i, file := range ingestedFiles {
				itemText := fmt.Sprintf("%d. %s", i+1, file.FilePath)
				dp.detailsList.AddItem(itemText, "", 0, nil)
			}
			dp.detailsList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
				parts := strings.SplitN(mainText, ". ", 2)
				if len(parts) == 2 {
					filepath := parts[1]
					dp.openValue(filepath)
				}
			})
			dp.detailsListFlex.AddItem(dp.detailsList, 0, 1, true)
		} else {
			dp.detailsListFlex.AddItem(dp.detailsListEmpty, 0, 1, true)
		}
	}

	dp.syncIngestedFilesFocus()
}

func (dp *DetailsPanel) focusIngestedFiles() {
	if dp.detailsListFlex.GetItemCount() > 0 {
		target := dp.detailsListFlex.GetItem(0)
		if target != nil {
			dp.app.tview.SetFocus(target)

			return
		}
	}

	dp.app.tview.SetFocus(dp.detailsListFlex)
}

func (dp *DetailsPanel) syncIngestedFilesFocus() {
	focus := dp.app.tview.GetFocus()
	if focus == dp.detailsList || focus == dp.detailsListEmpty || focus == dp.detailsListFlex {
		dp.focusIngestedFiles()
	}
}

func (dp *DetailsPanel) focusActiveEnvList() {
	if dp.app.envList.IsDockerActive() {
		if dp.app.envList.docker.GetItemCount() > 0 {
			dp.app.tview.SetFocus(dp.app.envList.docker)
		} else {
			dp.app.tview.SetFocus(dp.app.envList.dockerEmpty)
		}

		return
	}

	if dp.app.envList.k8s.GetItemCount() > 0 {
		dp.app.tview.SetFocus(dp.app.envList.k8s)
	} else {
		dp.app.tview.SetFocus(dp.app.envList.k8sEmpty)
	}
}

// CycleFocus cycles focus between buttons, grid, and list in the details view.
func (dp *DetailsPanel) CycleFocus() {
	focus := dp.app.tview.GetFocus()
	switch focus {
	case dp.deleteButton:
		switch {
		case len(dp.nameDirButtons) > 0:
			dp.app.tview.SetFocus(dp.nameDirButtons[0])
		case len(dp.detailsButtons) > 0:
			dp.app.tview.SetFocus(dp.detailsButtons[0])
		default:
			dp.focusIngestedFiles()
		}
	case dp.cleanButton:
		dp.app.tview.SetFocus(dp.deleteButton)
	case dp.updateButton:
		if dp.currentDetailsType == string(DockerKey) {
			dp.app.tview.SetFocus(dp.cleanButton)
		} else {
			dp.app.tview.SetFocus(dp.deleteButton)
		}
	case dp.renderButton:
		dp.app.tview.SetFocus(dp.updateButton)
	case dp.populateButton:
		dp.app.tview.SetFocus(dp.renderButton)
	case dp.detailsListFlex, dp.detailsList, dp.detailsListEmpty:
		dp.app.tview.SetFocus(dp.populateButton)
	default:
		// Check if it's a details button
		for i, btn := range dp.detailsButtons {
			if focus == btn {
				if i+1 < len(dp.detailsButtons) {
					dp.app.tview.SetFocus(dp.detailsButtons[i+1])
				} else {
					dp.focusIngestedFiles()
				}
				return
			}
		}
		// Check if it's a nameDir button
		for i, btn := range dp.nameDirButtons {
			if focus == btn {
				if i+1 < len(dp.nameDirButtons) {
					dp.app.tview.SetFocus(dp.nameDirButtons[i+1])
				} else {
					if len(dp.detailsButtons) > 0 {
						dp.app.tview.SetFocus(dp.detailsButtons[0])
					} else {
						dp.focusIngestedFiles()
					}
				}
				return
			}
		}
		// If not, start at the top
		dp.app.tview.SetFocus(dp.populateButton)
	}
}

// CycleFocusBackward cycles focus backward between buttons, grid, and list in the details view.
func (dp *DetailsPanel) CycleFocusBackward() {
	focus := dp.app.tview.GetFocus()
	switch focus {
	case dp.detailsList, dp.detailsListEmpty, dp.detailsListFlex:
		switch {
		case len(dp.detailsButtons) > 0:
			dp.app.tview.SetFocus(dp.detailsButtons[len(dp.detailsButtons)-1])
		case len(dp.nameDirButtons) > 0:
			dp.app.tview.SetFocus(dp.nameDirButtons[len(dp.nameDirButtons)-1])
		default:
			dp.app.tview.SetFocus(dp.deleteButton)
		}
	case dp.deleteButton:
		if dp.currentDetailsType == string(DockerKey) {
			dp.app.tview.SetFocus(dp.cleanButton)
		} else {
			dp.app.tview.SetFocus(dp.updateButton)
		}
	case dp.cleanButton:
		dp.app.tview.SetFocus(dp.updateButton)
	case dp.updateButton:
		dp.app.tview.SetFocus(dp.renderButton)
	case dp.renderButton:
		dp.app.tview.SetFocus(dp.populateButton)
	case dp.populateButton:
		dp.focusIngestedFiles()
	default:
		// Check if it's a details button
		for i, btn := range dp.detailsButtons {
			if focus == btn {
				if i > 0 {
					dp.app.tview.SetFocus(dp.detailsButtons[i-1])
				} else {
					if len(dp.nameDirButtons) > 0 {
						dp.app.tview.SetFocus(dp.nameDirButtons[len(dp.nameDirButtons)-1])
					} else {
						dp.app.tview.SetFocus(dp.deleteButton)
					}
				}
				return
			}
		}
		// Check if it's a nameDir button
		for i, btn := range dp.nameDirButtons {
			if focus == btn {
				if i > 0 {
					dp.app.tview.SetFocus(dp.nameDirButtons[i-1])
				} else {
					dp.app.tview.SetFocus(dp.deleteButton)
				}
				return
			}
		}
		// If not, start at the end
		dp.focusIngestedFiles()
	}
}

// SetupInput configures key handlers for the details panel.
func (dp *DetailsPanel) SetupInput() {
	handler := func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyEsc:
			dp.Clear()
			dp.focusActiveEnvList()
			return nil
		case event.Key() == tcell.KeyTab:
			dp.CycleFocus()
			return nil
		case event.Key() == tcell.KeyBacktab:
			dp.CycleFocusBackward()
			return nil
		case event.Key() == tcell.KeyEnter:
			return event // Let the table handle via SetSelectedFunc
		case event.Rune() == 'd':
			dp.app.showDeleteConfirm()
			return nil
		case event.Rune() == 'c':
			if dp.currentDetailsType == string(DockerKey) {
				dp.app.showCleanConfirm()
				return nil
			}
		case event.Rune() == 'u':
			dp.app.showUpdateForm()
			return nil
		case event.Rune() == 'r':
			dp.app.showRenderForm()
			return nil
		case event.Rune() == 'p':
			dp.app.showPopulateForm()
			return nil
		case event.Rune() == 'g':
			if dp.detailsShown && len(dp.currentDetailsRows) > 0 {
				dp.openValue(dp.currentDetailsRows[0].Value)
				return nil
			}
		case event.Rune() == 'b':
			if dp.detailsShown {
				if value, ok := dp.findDetailRowValue("Backoffice"); ok {
					dp.openValue(value)
				}
				return nil
			}
		case event.Rune() == 't':
			if dp.detailsShown {
				if value, ok := dp.findDetailRowValue("Grafana"); ok {
					dp.openValue(value)
				}
				return nil
			}
		case event.Rune() == 'a':
			if dp.detailsShown {
				if value, ok := dp.findDetailRowValue("API"); ok {
					dp.openValue(value)
				}
				return nil
			}
		case event.Rune() == 'G':
			if dp.detailsShown && len(dp.currentDetailsRows) > 0 {
				dp.app.copyToClipboardWithFeedback(dp.currentDetailsRows[0].Value, "Copied to clipboard", "Failed to copy to clipboard")
				return nil
			}
		case event.Rune() == 'B':
			if dp.detailsShown {
				if value, ok := dp.findDetailRowValue("Backoffice"); ok {
					dp.app.copyToClipboardWithFeedback(value, "Copied to clipboard", "Failed to copy to clipboard")
				}
				return nil
			}
		case event.Rune() == 'A':
			if dp.detailsShown {
				if value, ok := dp.findDetailRowValue("API"); ok {
					dp.app.copyToClipboardWithFeedback(value, "Copied to clipboard", "Failed to copy to clipboard")
				}
				return nil
			}
		case event.Rune() == 'y':
			if dp.app.tview.GetFocus() == dp.detailsList {
				index := dp.detailsList.GetCurrentItem()
				mainText, _ := dp.detailsList.GetItemText(index)
				parts := strings.SplitN(mainText, ". ", 2)
				if len(parts) == 2 {
					filepath := parts[1]
					dp.app.copyToClipboardWithFeedback(filepath, "Copied to clipboard", "Failed to copy to clipboard")
					return nil
				}
			}
		}
		return event
	}
	dp.details.SetInputCapture(handler)
	dp.detailsList.SetInputCapture(handler)
	dp.detailsListEmpty.SetInputCapture(handler)
	dp.detailsListFlex.SetInputCapture(handler)
}

func (dp *DetailsPanel) findDetailRowValue(label string) (string, bool) {
	for _, row := range dp.currentDetailsRows {
		if row.Label == label {
			return row.Value, true
		}
	}

	return "", false
}

func (dp *DetailsPanel) hasDetailRow(label string) bool {
	_, ok := dp.findDetailRowValue(label)
	return ok
}

// setupFocusHandlers configures visual feedback when components gain/lose focus.
func (dp *DetailsPanel) setupFocusHandlers() {
	dp.detailsList.SetFocusFunc(func() {
		updateListStyle(dp.detailsList, true)
	})
	dp.detailsList.SetBlurFunc(func() {
		updateListStyle(dp.detailsList, false)
	})
}

// createDetailsRows creates the grid rows for details.
func (dp *DetailsPanel) createDetailsRows(rows []DetailRow) {
	dp.createGridRows(dp.detailsGrid, rows, &dp.detailsButtons, "Environment URLs")
}

// openValue opens the given value (URL, directory, or file) using the appropriate command.
func (dp *DetailsPanel) openValue(value string) {
	if err := dp.app.openValue(value); err != nil {
		dp.app.ShowError(err.Error())
	}
}

// createGridRows creates the grid rows for details or name/dir
func (dp *DetailsPanel) createGridRows(grid *tview.Grid, rows []DetailRow, buttons *[]*tview.Button, header string) {
	grid.Clear()
	*buttons = nil

	numColumns := 3
	hasOpenButtons := false
	hasExtraButtons := false
	for _, row := range rows {
		if row.IncludeOpen {
			numColumns = 4
			hasOpenButtons = true
		}
		if row.ExtraAction != nil {
			hasExtraButtons = true
		}
	}
	if hasExtraButtons {
		numColumns++
	}

	// Calculate total rows: header + separator + (data rows + spacing rows)
	totalRows := 0
	if header != "" {
		totalRows += 2 // Header + separator
	}
	totalRows += len(rows)
	if len(rows) > 0 {
		totalRows += len(rows) - 1 // Add spacing rows between data rows (not after last)
	}

	rowHeights := make([]int, totalRows)
	for i := range rowHeights {
		rowHeights[i] = 1
	}
	grid.SetRows(rowHeights...)

	switch {
	case hasExtraButtons && hasOpenButtons:
		grid.SetColumns(15, 0, 17, 12, 10)
	case hasOpenButtons:
		grid.SetColumns(15, 0, 12, 10)
	case hasExtraButtons:
		grid.SetColumns(15, 0, 17, 12)
	default:
		grid.SetColumns(15, 0, 12)
	}

	rowIndex := 0
	if header != "" {
		headerTV := tview.NewTextView().
			SetDynamicColors(true).
			SetText(DefaultTheme.SecondaryTag("b") + header)
		headerTV.SetBorderPadding(0, 0, 1, 1)

		grid.AddItem(headerTV, rowIndex, 0, 1, numColumns, 0, 0, false)
		rowIndex++

		// use Box that fills available width for the separator
		separatorBox := tview.NewBox().
			SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
				// Draw separator line across full width
				style := tcell.StyleDefault.Foreground(DefaultTheme.Muted).Background(DefaultTheme.Background)
				for i := range width {
					screen.SetContent(x+i, y, '─', nil, style)
				}
				return x, y, width, height
			})

		grid.AddItem(separatorBox, rowIndex, 0, 1, numColumns, 0, 0, false)
		rowIndex++
	}

	for i, row := range rows {
		copyAction := func() {
			dp.app.copyToClipboardWithFeedback(row.Value, "Copied to clipboard", "Failed to copy to clipboard")
		}
		if row.OnCopy != nil {
			copyAction = row.OnCopy
		}

		openAction := func() {
			dp.openValue(row.Value)
		}
		if row.OnOpen != nil {
			openAction = row.OnOpen
		}

		labelTV := tview.NewTextView().
			SetDynamicColors(true).
			SetText(DefaultTheme.PrimaryTag("b") + row.Label)
		labelTV.SetBorderPadding(0, 0, 1, 1)

		valueTV := tview.NewTextView().
			SetText(row.Value).
			SetTextColor(DefaultTheme.OnSurface)
		valueTV.SetBorderPadding(0, 0, 1, 1)

		copyLabel := "Copy"
		openLabel := "Open ↗"
		if strings.HasPrefix(row.Value, "http://") || strings.HasPrefix(row.Value, "https://") {
			copyLabel = "Copy URL"
		}
		copyBtn := NewStyledButton(copyLabel, copyAction)

		// wrap button in a box with padding
		copyBtnBox := tview.NewFlex().SetDirection(tview.FlexColumn)
		copyBtnBox.AddItem(tview.NewBox(), 1, 0, false)
		copyBtnBox.AddItem(copyBtn, 0, 1, false)
		copyBtnBox.AddItem(tview.NewBox(), 1, 0, false)

		grid.AddItem(labelTV, rowIndex, 0, 1, 1, 0, 0, false)
		grid.AddItem(valueTV, rowIndex, 1, 1, 1, 0, 0, false)
		copyColumn := 2
		if hasExtraButtons {
			copyColumn++
			if row.ExtraAction != nil {
				button := NewStyledButton(row.ExtraActionLabel, row.ExtraAction)
				buttonBox := tview.NewFlex().SetDirection(tview.FlexColumn)
				buttonBox.AddItem(tview.NewBox(), 1, 0, false)
				buttonBox.AddItem(button, 0, 1, false)
				buttonBox.AddItem(tview.NewBox(), 1, 0, false)
				grid.AddItem(buttonBox, rowIndex, 2, 1, 1, 0, 0, false)
				*buttons = append(*buttons, button)
			}
		}
		grid.AddItem(copyBtnBox, rowIndex, copyColumn, 1, 1, 0, 0, false)

		*buttons = append(*buttons, copyBtn)

		if row.IncludeOpen {
			openBtn := NewStyledButton(openLabel, openAction)

			// wrap button in a box with padding
			openBtnBox := tview.NewFlex().SetDirection(tview.FlexColumn)
			openBtnBox.AddItem(tview.NewBox(), 1, 0, false)
			openBtnBox.AddItem(openBtn, 0, 1, false)
			openBtnBox.AddItem(tview.NewBox(), 1, 0, false)

			grid.AddItem(openBtnBox, rowIndex, copyColumn+1, 1, 1, 0, 0, false)
			*buttons = append(*buttons, openBtn)
		}

		rowIndex++

		// add spacing row between items but not after the last item
		if i < len(rows)-1 {
			spacingBox := tview.NewBox()
			grid.AddItem(spacingBox, rowIndex, 0, 1, numColumns, 0, 0, false)
			rowIndex++
		}
	}
}
