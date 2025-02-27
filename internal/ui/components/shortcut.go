package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ShortcutBar() *tview.TextView {
	text := tview.NewTextView().
		SetText(" <TAB> Switch Panel \n  q Quit ? Help \n  d Details  b Go back ").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorYellow)
	return text
}
