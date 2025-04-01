package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	Grid            *tview.Grid
	backgroundColor tcell.Color
	textColor       tcell.Color
	headerColor     tcell.Color
	keyColor        tcell.Color
}

type HelpSection struct {
	Title     string
	Shortcuts map[string]string
}

func NewHelpView() *HelpView {
	defaultBgColor := tcell.ColorBlack
	defaultTextColor := tcell.ColorWhite
	defaultHeaderColor := tcell.ColorYellow
	defaultKeyColor := tcell.NewHexColor(0x017be9)

	helpView := &HelpView{
		backgroundColor: defaultBgColor,
		textColor:       defaultTextColor,
		headerColor:     defaultHeaderColor,
		keyColor:        defaultKeyColor,
	}

	helpView.Grid = helpView.renderHelp()

	return helpView
}

func (h *HelpView) GenerateSections() []HelpSection {
	return []HelpSection{
		{
			Title: "GENERAL",
			Shortcuts: map[string]string{
				"q": "Close TUI application",
				"?": "Show/hide this help",
				"b": "Go back to previous screen",
				"/": "Search in current view",
				":": "Alternative search key",
				"I": "Return to instance selection",
			},
		},
		{
			Title: "APPLICATIONS",
			Shortcuts: map[string]string{
				"R":     "Refresh all applications",
				"r":     "Refresh selected application",
				"S":     "Sync selected application",
				"D":     "Delete selected application",
				"↑/↓":   "Navigate applications list",
				"Enter": "Open application resources",
			},
		},
		{
			Title: "FILTERING",
			Shortcuts: map[string]string{
				"f, F": "Show filter menu",
				"h, H": "Toggle Healthy filter",
				"d":    "Toggle Degraded filter",
				"p, P": "Toggle Progressing filter",
				"s":    "Toggle Synced filter",
				"o, O": "Toggle OutOfSync filter",
				"c, C": "Clear all filters",
			},
		},
		{
			Title: "RESOURCES",
			Shortcuts: map[string]string{
				"t": "Toggle resource tree expansion",
				"d": "Filter by Deployments",
				"s": "Filter by Services",
				"i": "Filter by Ingress",
				"c": "Filter by ConfigMaps",
			},
		},
	}
}

func (h *HelpView) renderHelp() *tview.Grid {

	h.Grid = tview.NewGrid().
		SetBorders(false).
		SetRows(0, 2)

	h.Grid.SetTitle(fmt.Sprintf(" [%s::b]ArguTUI - KEYBOARD SHORTCUTS[-:-:-] ", h.headerColor)).
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter).
		SetBackgroundColor(h.backgroundColor).
		SetTitleColor(h.headerColor)

	sections := h.GenerateSections()
	for i := 0; i < len(sections); i++ {
		view := tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignLeft).
			SetWrap(true)

		var sectionText strings.Builder

		sectionText.WriteString(fmt.Sprintf("[%s::b]%s:[-:-:-]\n", h.headerColor, sections[i].Title))

		for key, desc := range sections[i].Shortcuts {
			sectionText.WriteString(fmt.Sprintf("[%s]%s[%s] %s\n",
				h.keyColor.String(),
				key,
				h.textColor.String(),
				desc))
		}

		view.SetText(sectionText.String())
		h.Grid.AddItem(view, 0, i, 1, 1, 0, 0, false)
	}

	closeHintView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("\n[" + h.keyColor.String() + "]Press '?' to close this help screen[-:-:-]")

	h.Grid.AddItem(closeHintView, 1, 0, 1, len(sections), 0, 0, false)

	return h.Grid
}

func (h *HelpView) GetInputCapture(onClose func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '?' {
			if onClose != nil {
				onClose()
			}
			return nil
		}
		return event
	}
}
