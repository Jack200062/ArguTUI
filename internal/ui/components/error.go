package components

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ErrorModal(errorTitle, errorDetails string, onClose func()) *tview.Modal {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("%s\n\n%v", errorTitle, errorDetails)).
		// Multiple options?
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if onClose != nil {
				onClose()
			}
		})
	modal.SetBackgroundColor(tcell.ColorMidnightBlue).SetTextColor(tcell.ColorWhite)
	return modal
}
