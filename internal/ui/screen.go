package ui

import "github.com/rivo/tview"

type Screen interface {
	Init() tview.Primitive
	Name() string
}
