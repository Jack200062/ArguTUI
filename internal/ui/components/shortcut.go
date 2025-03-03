package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ShortcutGroup struct {
	Title     string
	Shortcuts map[string]string
}

type ShortcutBar struct {
	view             *tview.Flex
	backgroundColor  tcell.Color
	shortcutKeyColor tcell.Color
	groups           []ShortcutGroup
}

func NewShortcutBar(backgroundColor, shortcutKeyColor tcell.Color) *ShortcutBar {
	return &ShortcutBar{
		backgroundColor:  backgroundColor,
		shortcutKeyColor: shortcutKeyColor,
		groups:           []ShortcutGroup{},
	}
}

func (s *ShortcutBar) AddGroup(title string, shortcuts map[string]string) {
	s.groups = append(s.groups, ShortcutGroup{
		Title:     title,
		Shortcuts: shortcuts,
	})
}

func (s *ShortcutBar) Init() tview.Primitive {
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow)
	flex.SetBackgroundColor(s.backgroundColor)

	for _, group := range s.groups {
		var textBuilder strings.Builder

		if group.Title != "" {
			textBuilder.WriteString(fmt.Sprintf("[yellow::b]%s[-:-:-]: ", group.Title))
		}

		for key, description := range group.Shortcuts {
			textBuilder.WriteString(fmt.Sprintf("[%s]%s[gray] %s  ",
				s.shortcutKeyColor, key, description))
		}

		row := tview.NewTextView().
			SetDynamicColors(true).
			SetText(textBuilder.String()).
			SetTextAlign(tview.AlignLeft)

		row.SetBackgroundColor(s.backgroundColor)
		flex.AddItem(row, 1, 0, false)
	}

	s.view = flex
	return flex
}

func NewHorizontalShortcutBar(shortcuts map[string]string, backgroundColor, shortcutKeyColor tcell.Color) *tview.TextView {
	var textBuilder strings.Builder

	for key, description := range shortcuts {
		textBuilder.WriteString(fmt.Sprintf("[%s]%s[gray] %s  ",
			shortcutKeyColor, key, description))
	}

	shortcutsView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(textBuilder.String()).
		SetTextAlign(tview.AlignCenter)
	shortcutsView.SetBackgroundColor(backgroundColor)

	return shortcutsView
}

func ShortcutBar_Old() *tview.TextView {
	text := tview.NewTextView().
		SetText(" q Quit ? Help \n  d Details  b Go back ").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorYellow)
	return text
}
