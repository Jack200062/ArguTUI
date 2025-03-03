package filters

import (
	"fmt"
	"unicode"

	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MenuOptions struct {
	Title          string
	ActiveFilters  string
	Projects       []string
	HealthStatuses []string
	SyncStatuses   []string
	ProjectFilter  string
	HealthFilter   string
	SyncFilter     string
}

type FilterMenuItem struct {
	Name     string
	Shortcut rune
	Action   func()
}

func ShowFilterMenu(
	app *tview.Application,
	options MenuOptions,
	returnApp tview.Primitive,
	onProjectFilterSelect func(string),
	onHealthFilterSelect func(string),
	onSyncFilterSelect func(string),
	onClearFilters func(),
) {
	theme := DefaultTheme()

	modal := NewBaseFilterModal(app, "FILTER APPLICATIONS", returnApp, theme)

	if options.Title == "" {
		options.Title = "FILTER APPLICATIONS"
	}

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow)

	if options.ActiveFilters != "" {
		filtersText := tview.NewTextView().
			SetDynamicColors(true).
			SetText("Current filters: " + StyleText(options.ActiveFilters, theme.HeaderText)).
			SetTextAlign(tview.AlignCenter)
		filtersText.SetBackgroundColor(theme.Background)
		flex.AddItem(filtersText, 1, 0, false)
	}

	shortcutBar := components.NewShortcutBar(theme.Background, theme.ShortcutKey)

	shortcutBar.AddGroup("General", map[string]string{
		"a":     "All (clear all filters)",
		"Esc":   "Cancel",
		"Enter": "Apply",
	})

	healthShortcuts := StandardHealthShortcuts()
	displayHealthShortcuts := map[string]string{}
	for status, shortcut := range healthShortcuts {
		for _, availableStatus := range options.HealthStatuses {
			if availableStatus == status {
				displayHealthShortcuts[string(shortcut)] = status
				break
			}
		}
	}
	if len(displayHealthShortcuts) > 0 {
		shortcutBar.AddGroup("Health", displayHealthShortcuts)
	}

	syncShortcuts := StandardSyncShortcuts()
	displaySyncShortcuts := map[string]string{}
	for status, shortcut := range syncShortcuts {
		for _, availableStatus := range options.SyncStatuses {
			if availableStatus == status {
				displaySyncShortcuts[string(shortcut)] = status
				break
			}
		}
	}
	if len(displaySyncShortcuts) > 0 {
		shortcutBar.AddGroup("Sync", displaySyncShortcuts)
	}

	shortcutBarPrimitive := shortcutBar.Init()
	flex.AddItem(shortcutBarPrimitive, 0, 1, false)

	buttons := []FilterMenuItem{
		{
			Name:     "Clear All Filters",
			Shortcut: 'a',
			Action: func() {
				if onClearFilters != nil {
					onClearFilters()
				}
				modal.Close()
			},
		},
		{
			Name:     "Project Filter",
			Shortcut: 'p',
			Action: func() {
				modal.Close()
				ShowProjectFilter(app, options.Projects, options.ProjectFilter, returnApp, func(result FilterResult) {
					if !result.Canceled && onProjectFilterSelect != nil {
						onProjectFilterSelect(result.Value)
					}
				})
			},
		},
		{
			Name:     "Health Filter",
			Shortcut: 'h',
			Action: func() {
				modal.Close()
				ShowHealthFilter(app, options.HealthStatuses, options.HealthFilter, returnApp, func(result FilterResult) {
					if !result.Canceled && onHealthFilterSelect != nil {
						onHealthFilterSelect(result.Value)
					}
				})
			},
		},
		{
			Name:     "Sync Filter",
			Shortcut: 's',
			Action: func() {
				modal.Close()
				ShowSyncFilter(app, options.SyncStatuses, options.SyncFilter, returnApp, func(result FilterResult) {
					if !result.Canceled && onSyncFilterSelect != nil {
						onSyncFilterSelect(result.Value)
					}
				})
			},
		},
	}

	buttonsList := tview.NewList().
		SetSelectedBackgroundColor(theme.Selection).
		SetSelectedTextColor(theme.SelectionText).
		SetMainTextColor(theme.Text).
		SetHighlightFullLine(true)
	buttonsList.SetBorder(false)
	buttonsList.SetBackgroundColor(theme.Background)

	for _, button := range buttons {
		localButton := button
		buttonsList.AddItem(
			fmt.Sprintf("%s", localButton.Name),
			"",
			localButton.Shortcut,
			func() {
				localButton.Action()
			},
		)
	}

	flex.AddItem(buttonsList, 0, 2, true)

	customHandler := func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'a' || event.Rune() == 'A' {
			if onClearFilters != nil {
				onClearFilters()
			}
			modal.Close()
			return nil
		}

		for status, shortcut := range healthShortcuts {
			if event.Rune() == shortcut || event.Rune() == unicode.ToUpper(shortcut) {
				for _, availableStatus := range options.HealthStatuses {
					if availableStatus == status {
						modal.Close()
						if onHealthFilterSelect != nil {
							onHealthFilterSelect(status)
						}
						return nil
					}
				}
			}
		}

		for status, shortcut := range syncShortcuts {
			if event.Rune() == shortcut || event.Rune() == unicode.ToUpper(shortcut) {
				for _, availableStatus := range options.SyncStatuses {
					if availableStatus == status {
						modal.Close()
						if onSyncFilterSelect != nil {
							onSyncFilterSelect(status)
						}
						return nil
					}
				}
			}
		}

		for _, button := range buttons {
			if event.Rune() == button.Shortcut || event.Rune() == unicode.ToUpper(button.Shortcut) {
				button.Action()
				return nil
			}
		}

		return event
	}

	buttonsList.SetInputCapture(modal.GetInputCapture(customHandler))

	modal.SetContent(flex).
		SetDoneFunc(func(result FilterResult) {
			if result.Canceled {
				modal.Close()
			}
		}).
		Show()
}
