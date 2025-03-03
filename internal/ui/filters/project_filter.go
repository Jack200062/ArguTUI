package filters

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ShowProjectFilter(
	app *tview.Application,
	projects []string,
	currentFilter string,
	returnApp tview.Primitive,
	onDone FilterHandler,
) {
	theme := DefaultTheme()

	modal := NewBaseFilterModal(app, "SELECT PROJECT", returnApp, theme)

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

	for i, project := range projects {
		localProject := project
		var shortcut rune
		if i < 9 {
			shortcut = rune('1' + i)
		} else if i < 36 {
			shortcut = rune('a' + i - 9)
		} else {
			shortcut = 0
		}

		displayText := project
		if currentFilter == project {
			displayText = "▶ " + project
		}

		list.AddItem(displayText, "", shortcut, func() {
			if onDone != nil {
				onDone(FilterResult{Value: localProject})
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

		if event.Rune() >= '1' && event.Rune() <= '9' {
			index := int(event.Rune()-'1') + 1
			if index < list.GetItemCount() {
				list.SetCurrentItem(index)
				return nil
			}
		}

		if event.Rune() >= 'a' && event.Rune() <= 'z' {
			index := int(event.Rune() - 'a' + 10)
			if index < list.GetItemCount() {
				list.SetCurrentItem(index)
				return nil
			}
		}

		return event
	}

	list.SetInputCapture(modal.GetInputCapture(customHandler))
	modal.SetContent(list).Show()
}
