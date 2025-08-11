package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	View            *tview.TextView
	backgroundColor tcell.Color
	textColor       tcell.Color
	headerColor     tcell.Color
	keyColor        tcell.Color
}

type HelpSection struct {
	Title     string
	Shortcuts map[string]string
}

func NewHelpView(options ...func(*HelpView)) *HelpView {
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

	for _, option := range options {
		option(helpView)
	}

	helpView.createView()

	helpView.View.SetText(helpView.RenderHelp())

	return helpView
}

func WithBackgroundColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.backgroundColor = color
	}
}

func WithTextColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.textColor = color
	}
}

func WithHeaderColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.headerColor = color
	}
}

func WithKeyColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.keyColor = color
	}
}

func (h *HelpView) createView() {
	h.View = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetWrap(true)
	h.View.SetBorder(true).
		SetTitle(" Help ").
		SetTitleAlign(tview.AlignCenter)

	h.View.SetBackgroundColor(h.backgroundColor)
	h.View.SetTextColor(h.textColor)
	h.View.SetBorderColor(h.keyColor)
	h.View.SetTitleColor(h.headerColor)
}

func (h *HelpView) GenerateSections() []HelpSection {
	return []HelpSection{
		{
			Title: "GENERAL (k9s-like)",
			Shortcuts: map[string]string{
				"q":   "Quit",
				"?":   "Help",
				"esc": "Cancel/Close",
				"/":   "Filter/Search",
				":":   "Command mode",
				"I":   "Instance selection",
			},
		},
		{
			Title: "APPLICATIONS",
			Shortcuts: map[string]string{
				"j/k":      "Down/Up",
				"g/G":      "Top/Bottom",
				"ctrl-d/u": "Half page +/-",
				"r":        "Refresh (all)",
				"enter":    "Open resources",
				"ctrl-k":   "Delete app",
			},
		},
		{
			Title: "FILTERING",
			Shortcuts: map[string]string{
				"f": "Filter menu",
				"c": "Clear all filters",
				"h": "Filter Healthy",
				"p": "Filter Progressing",
				"o": "Filter OutOfSync",
			},
		},
		{
			Title: "RESOURCES (k9s-like)",
			Shortcuts: map[string]string{
				"j/k":      "Down/Up",
				"g/G":      "Top/Bottom",
				"ctrl-d/u": "Half page +/-",
				"t":        "Toggle expand all",
				"/":        "Filter/Search",
				"f":        "Filter menu",
				"esc":      "Back",
                "e":        "Edit in $EDITOR (vim)",
				"d":        "Filter Deployments",
				"s":        "Filter Services",
				"i":        "Filter Ingress",
				"c":        "Filter ConfigMaps",
			},
		},
	}
}

func (h *HelpView) RenderHelp() string {
	var helpText strings.Builder

	helpText.WriteString(fmt.Sprintf("[%s::b]ArguTUI - KEYBOARD SHORTCUTS[-:-:-]\n\n", h.headerColor))

	for _, section := range h.GenerateSections() {
		helpText.WriteString(fmt.Sprintf("[%s::b]%s:[-:-:-]\n", h.headerColor, section.Title))

		for key, desc := range section.Shortcuts {
			helpText.WriteString(fmt.Sprintf("[%s]%s[%s] %s\n",
				h.keyColor.String(),
				key,
				h.textColor.String(),
				desc))
		}
		helpText.WriteString("\n")
	}

	helpText.WriteString("\n[" + h.keyColor.String() + "]Press '?' to close this help screen[-:-:-]")

	return helpText.String()
}

func (h *HelpView) Render() *tview.TextView {
	h.View.SetText(h.RenderHelp())
	return h.View
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
