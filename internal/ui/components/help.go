package components

import (
	"github.com/rivo/tview"
)

type HelpView struct {
	*tview.Flex
}

func NewHelpView() *HelpView {
	flex := tview.NewFlex().SetDirection(tview.FlexColumn)

	resourceText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow::b]RESOURCE[-:-:-]\n" +
			" [blue]<tab>[white] next resource\n" +
			" [blue]<enter>[white] open resource\n" +
			" [blue]g[white] top of list\n" +
			" ...")
	generalText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow::b]GENERAL[-:-:-]\n" +
			" [blue]q[white] quit\n" +
			" [blue]?[white] show help\n" +
			" [blue]d[white] details\n" +
			" ...")
	hotkeysText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow::b]HOTKEYS[-:-:-]\n" +
			" [blue]b[white] go back\n" +
			" [blue]/ or :[white] open search\n" +
			" [blue]<tab>[white] switch focus\n" +
			" ...")

	flex.AddItem(resourceText, 0, 1, false).
		AddItem(generalText, 0, 1, false).
		AddItem(hotkeysText, 0, 1, false)

	return &HelpView{Flex: flex}
}
