package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SimpleSearchBar struct {
	InputField *tview.InputField
}

func NewSimpleSearchBar(label string, fieldWidth int) *SimpleSearchBar {
	input := tview.NewInputField().
		SetLabel(label).
		SetFieldWidth(fieldWidth).
		SetFieldStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack))
	return &SimpleSearchBar{InputField: input}
}
