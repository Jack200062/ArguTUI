package filters

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func GetHealthStatusColor(status string) tcell.Color {
	switch strings.ToLower(status) {
	case "healthy":
		return tcell.ColorGreen
	case "progressing":
		return tcell.ColorYellow
	case "degraded":
		return tcell.ColorRed
	case "suspended":
		return tcell.ColorBlue
	default:
		return tcell.ColorWhite
	}
}

func StandardHealthShortcuts() map[string]rune {
	return map[string]rune{
		"Healthy":     'h',
		"Progressing": 'p',
		"Degraded":    'd',
		"Suspended":   's',
		"Unknown":     'u',
	}
}

func ShowHealthFilter(
	app *tview.Application,
	statuses []string,
	currentFilter string,
	returnApp tview.Primitive,
	onDone FilterHandler,
) {
	theme := DefaultTheme()

	modal := NewBaseFilterModal(app, "SELECT HEALTH STATUS", returnApp, theme)

	standardShortcuts := StandardHealthShortcuts()

	list := tview.NewList().
		SetSelectedBackgroundColor(theme.Selection).
		SetSelectedTextColor(theme.SelectionText).
		SetMainTextColor(theme.Text).
		SetHighlightFullLine(true)
	list.SetBorder(false)
	list.SetBackgroundColor(theme.Background)

	list.AddItem("All (clear filter)", "", 'a', func() {
		if onDone != nil {
			onDone(FilterResult{Value: ""})
		}
		modal.Close()
	})

	for _, status := range statuses {
		localStatus := status
		shortcut, hasShortcut := standardShortcuts[status]
		if !hasShortcut {
			shortcut = 0
		}

		displayText := StyleText(status, GetHealthStatusColor(status))
		if currentFilter == status {
			displayText = "▶ " + displayText
		}

		list.AddItem(displayText, "", shortcut, func() {
			if onDone != nil {
				onDone(FilterResult{Value: localStatus})
			}
			modal.Close()
		})
	}

	customHandler := func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'a' || event.Rune() == 'A' {
			if onDone != nil {
				onDone(FilterResult{Value: ""})
			}
			modal.Close()
			return nil
		}

		for status, sc := range standardShortcuts {
			if event.Rune() == sc || event.Rune() == unicode.ToUpper(sc) {
				for i, availableStatus := range statuses {
					if availableStatus == status {
						list.SetCurrentItem(i + 1)
						if onDone != nil {
							onDone(FilterResult{Value: status})
						}
						modal.Close()
						return nil
					}
				}
			}
		}

		return event
	}

	list.SetInputCapture(modal.GetInputCapture(customHandler))
	modal.SetContent(list).Show()
}
