package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilterFunc func(query string) []interface{}

type SearchBar struct {
	InputField *tview.InputField
	Filter     FilterFunc
	OnDone     func(filtered []interface{})
}

func NewSearchBar(label string, fieldWidth int) *SearchBar {
	input := tview.NewInputField().
		SetLabel(label).
		SetFieldWidth(fieldWidth).
		SetFieldStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack))
	sb := &SearchBar{
		InputField: input,
	}
	sb.InputField.SetDoneFunc(func(key tcell.Key) {
		if sb.Filter != nil && sb.OnDone != nil {
			filtered := sb.Filter(sb.InputField.GetText())
			sb.OnDone(filtered)
		}
	})
	sb.InputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})
	return sb
}
