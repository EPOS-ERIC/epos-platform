package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/cli/config"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker"
	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FocusButton represents the button to focus after UI rebuild.
type FocusButton string

const (
	FocusButtonBrowse FocusButton = "browse"
	FocusButtonFiles  FocusButton = "files"
	FocusButtonDirs   FocusButton = "dirs"
)

// populateState holds the state of the populate form.
type populateState struct {
	paths       []string
	examples    bool
	token       string
	inputs      []*tview.InputField
	focusButton FocusButton // "browse", "files", "dirs", or ""
}

// showPopulateForm displays the dynamic populate form.
func (a *App) showPopulateForm(envName string, isDocker bool, k8sContext string) {
	a.PushFocus()

	if envName == "" {
		return
	}
	var requiresToken bool
	if isDocker {
		env, err := docker.GetEnv(envName)
		if err != nil {
			a.ShowError(fmt.Sprintf("Error loading Docker environment: %v", err))
			return
		}
		requiresToken = !env.Components.AAIService.Enabled
	} else {
		env, err := k8s.GetEnv(envName, k8sContext)
		if err != nil {
			a.ShowError(fmt.Sprintf("Error loading Kubernetes environment: %v", err))
			return
		}
		requiresToken = !env.Components.AAIService.Enabled
	}

	a.UpdateFooter(PopulateFormKey)

	state := &populateState{
		paths:    []string{""},
		examples: false,
	}

	formFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	formFlex.SetBorder(true).
		SetBorderColor(DefaultTheme.Primary).
		SetTitle(fmt.Sprintf(" [::b]Populate: %s ", envName)).
		SetTitleColor(DefaultTheme.Secondary)

	var rebuildUI func()
	rebuildUI = func() {
		formFlex.Clear()
		state.inputs = nil

		updatePathsAndRebuild := func(selectedPaths []string, focusBtn FocusButton) {
			state.paths = appendUnique(state.paths, selectedPaths)
			// remove leading empty placeholders if there are actual paths
			for i, p := range state.paths {
				if p != "" {
					state.paths = state.paths[i:]
					break
				}
			}
			// if all are empty, keep one empty for input
			if len(state.paths) == 0 {
				state.paths = []string{""}
			}
			state.focusButton = focusBtn
			rebuildUI()
		}

		for i, path := range state.paths {
			idx := i
			input := NewStyledInputField(fmt.Sprintf("Path %d ", i+1), path).
				SetChangedFunc(func(text string) {
					if idx < len(state.paths) {
						state.paths[idx] = text
					}
				})

			input.SetBorderPadding(0, 0, 1, 1)
			size := 1

			removeBtn := tview.NewButton("✗")
			removeBtn.SetStyle(tcell.StyleDefault.Background(DefaultTheme.Destructive).Foreground(DefaultTheme.OnDestructive))
			removeBtn.SetActivatedStyle(tcell.StyleDefault.Background(DefaultTheme.Secondary).Foreground(DefaultTheme.OnSecondary))
			removeBtn.SetSelectedFunc(func() {
				// remove the path at this index
				if idx < len(state.paths) {
					state.paths = append(state.paths[:idx], state.paths[idx+1:]...)
					// Ensure we always have at least one empty path
					if len(state.paths) == 0 {
						state.paths = []string{""}
					}
					rebuildUI()
					// focus the first input after removal
					if len(state.inputs) > 0 {
						focusIdx := idx
						if focusIdx >= len(state.inputs) {
							focusIdx = len(state.inputs) - 1
						}
						a.tview.SetFocus(state.inputs[focusIdx])
					}
				}
			})

			pathRow := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(input, 0, 1, true).
				AddItem(removeBtn, 3, 0, false)

			// initial top padding
			if i == 0 {
				formFlex.AddItem(tview.NewBox(), 1, 0, false)
			}
			formFlex.AddItem(pathRow, size, 0, true).
				AddItem(tview.NewBox(), 1, 0, false)

			state.inputs = append(state.inputs, input)
		}

		addPathBtn := NewStyledInactiveButton("Add Path", func() {
			state.paths = append(state.paths, "")
			rebuildUI()
			if len(state.inputs) > 0 {
				a.tview.SetFocus(state.inputs[len(state.inputs)-1])
			}
		})

		var browseBtn *tview.Button
		var browseFilesBtn *tview.Button
		var browseDirsBtn *tview.Button
		if a.config.TUI.FilePickerMode == config.FilePickerModeTUI {
			browseBtn = NewStyledInactiveButton("Browse", func() {
				var currentPaths []string
				for _, p := range state.paths {
					if strings.TrimSpace(p) != "" {
						currentPaths = append(currentPaths, p)
					}
				}
				startPath := ""
				if len(currentPaths) > 0 {
					startPath = currentPaths[0]
				}
				a.showTUIFilePicker(startPath, currentPaths, func(selectedPaths []string) {
					updatePathsAndRebuild(selectedPaths, FocusButtonBrowse)
				})
			})
		} else {
			browseFilesBtn = NewStyledInactiveButton("Browse Files", func() {
				a.showFilePickerNative(false, func(selectedPaths []string) {
					updatePathsAndRebuild(selectedPaths, FocusButtonFiles)
				})
			})

			browseDirsBtn = NewStyledInactiveButton("Browse Dirs", func() {
				a.showFilePickerNative(true, func(selectedPaths []string) {
					updatePathsAndRebuild(selectedPaths, FocusButtonDirs)
				})
			})
		}

		// set focus based on pending focus button
		switch state.focusButton {
		case FocusButtonBrowse:
			if browseBtn != nil {
				a.tview.SetFocus(browseBtn)
			}
		case FocusButtonFiles:
			if browseFilesBtn != nil {
				a.tview.SetFocus(browseFilesBtn)
			}
		case FocusButtonDirs:
			if browseDirsBtn != nil {
				a.tview.SetFocus(browseDirsBtn)
			}
		}
		state.focusButton = ""

		checkbox := tview.NewCheckbox().
			SetLabel("Populate Examples ").
			SetChecked(state.examples).
			SetChangedFunc(func(checked bool) {
				state.examples = checked
			})
		checkbox.SetLabelColor(DefaultTheme.Secondary).
			SetFieldBackgroundColor(DefaultTheme.Surface).
			SetFieldTextColor(DefaultTheme.Secondary).
			SetBorderPadding(0, 0, 1, 1)

		label := "Bearer token (optional) "
		if requiresToken {
			label = "Bearer token (required*) "
		}
		tokenInput := NewStyledInputField(label, state.token).
			SetMaskCharacter('*').
			SetChangedFunc(func(text string) { state.token = text })
		tokenInput.SetBorderPadding(0, 0, 1, 1)

		populateBtn := NewStyledButton("Populate", func() {
			if requiresToken && strings.TrimSpace(state.token) == "" && strings.TrimSpace(os.Getenv("EPOS_BEARER_TOKEN")) == "" {
				a.ShowError("A bearer token or EPOS_BEARER_TOKEN is required for external AAI")
				return
			}
			a.handlePopulate(envName, k8sContext, state, isDocker)
		})

		cancelBtn := NewStyledButton("Cancel", func() {
			a.ResetToHome(ResetOptions{
				PageNames:    []string{"populate"},
				RefreshFiles: true,
				RestoreFocus: true,
			})
		})

		controlsFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox(), 0, 1, false)
		if a.config.TUI.FilePickerMode == config.FilePickerModeTUI {
			controlsFlex.AddItem(browseBtn, 16, 1, false).
				AddItem(tview.NewBox(), 2, 0, false).
				AddItem(addPathBtn, 12, 1, false)
		} else {
			controlsFlex.AddItem(browseDirsBtn, 15, 1, false).
				AddItem(tview.NewBox(), 2, 0, false).
				AddItem(browseFilesBtn, 16, 1, false).
				AddItem(tview.NewBox(), 2, 0, false).
				AddItem(addPathBtn, 12, 1, false)
		}
		controlsFlex.AddItem(tview.NewBox(), 0, 1, false)

		controls := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(controlsFlex, 1, 0, false).
			AddItem(tview.NewBox(), 1, 0, false).
			AddItem(checkbox, 1, 0, false)
		if tokenInput != nil {
			controls.AddItem(tview.NewBox(), 1, 0, false).
				AddItem(tokenInput, 1, 0, false)
		}
		controls = controls.
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tview.NewBox(), 0, 1, false).
				AddItem(populateBtn, 12, 1, false).
				AddItem(tview.NewBox(), 2, 0, false).
				AddItem(cancelBtn, 10, 1, false).
				AddItem(tview.NewBox(), 0, 1, false), 1, 0, false)

		formFlex.AddItem(controls, 0, 1, false)

		var allFocusable []tview.Primitive

		// extract input fields and buttons
		for i := range formFlex.GetItemCount() {
			item := formFlex.GetItem(i)
			if pathRow, ok := item.(*tview.Flex); ok && pathRow.GetItemCount() == 2 {
				input := pathRow.GetItem(0).(*tview.InputField)

				allFocusable = append(allFocusable, input)

				if input.GetText() != "" {
					allFocusable = append(allFocusable, pathRow.GetItem(1))
				}
			}
		}

		if a.config.TUI.FilePickerMode == config.FilePickerModeTUI {
			allFocusable = append(allFocusable, browseBtn, addPathBtn, checkbox)
		} else {
			allFocusable = append(allFocusable, browseDirsBtn, browseFilesBtn, addPathBtn, checkbox)
		}
		if tokenInput != nil {
			allFocusable = append(allFocusable, tokenInput)
		}
		allFocusable = append(allFocusable, populateBtn, cancelBtn)

		formFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				a.ResetToHome(ResetOptions{
					PageNames:    []string{"populate"},
					RefreshFiles: true,
					RestoreFocus: true,
				})
				return nil
			}
			if event.Key() == tcell.KeyTab {
				current := a.tview.GetFocus()
				for i, p := range allFocusable {
					if p == current {
						next := i + 1
						if next >= len(allFocusable) {
							next = 0
						}
						a.tview.SetFocus(allFocusable[next])
						return nil
					}
				}
				if len(allFocusable) > 0 {
					a.tview.SetFocus(allFocusable[0])
				}
				return nil
			}
			if event.Key() == tcell.KeyBacktab {
				current := a.tview.GetFocus()
				for i, p := range allFocusable {
					if p == current {
						prev := i - 1
						if prev < 0 {
							prev = len(allFocusable) - 1
						}
						a.tview.SetFocus(allFocusable[prev])
						return nil
					}
				}
			}
			return event
		})
	}

	rebuildUI()

	a.pages.AddPage("populate", CenterPrimitiveFixed(formFlex, 65, 20), true, true)
	a.currentPage = "populate"
	if len(state.inputs) > 0 {
		a.tview.SetFocus(state.inputs[0])
	}
}

// handlePopulate validates the form and starts population.
func (a *App) handlePopulate(envName, context string, state *populateState, isDocker bool) {
	var validPaths []string
	for _, p := range state.paths {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			validPaths = append(validPaths, trimmed)
		}
	}

	a.showPopulateProgress(envName, context, validPaths, state.examples, isDocker, state.token)
}

// showPopulateProgress displays the populate progress with live output.
func (a *App) showPopulateProgress(envName, context string, paths []string, examples, isDocker bool, token string) {
	a.RunBackgroundTask(TaskOptions{
		Operation: "Populate",
		EnvName:   envName,
		IsDocker:  isDocker,
		Task: func() (string, error) {
			var err error
			if isDocker {
				_, err = docker.Populate(docker.PopulateOpts{
					Name:             envName,
					TTLDirs:          paths,
					PopulateExamples: examples,
					BearerToken:      token,
				})
			} else {
				_, err = k8s.Populate(k8s.PopulateOpts{
					Name:             envName,
					Context:          context,
					TTLDirs:          paths,
					PopulateExamples: examples,
					BearerToken:      token,
				})
			}
			if err != nil {
				display.Error("%v", err)
			}
			return "", err
		},
	})
}

// appendUnique merges two slices, removing duplicates while preserving order.
func appendUnique(existing, new []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(existing)+len(new))

	for _, p := range existing {
		if !seen[p] {
			result = append(result, p)
			seen[p] = true
		}
	}

	for _, p := range new {
		if !seen[p] {
			result = append(result, p)
			seen[p] = true
		}
	}

	return result
}
