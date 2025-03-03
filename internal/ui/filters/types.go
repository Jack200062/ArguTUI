package filters

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilterType string

const (
	ProjectFilter FilterType = "project"
	HealthFilter  FilterType = "health"
	SyncFilter    FilterType = "sync"
)

type FilterState struct {
	Type  FilterType
	Value string
}

type FilterResult struct {
	Canceled bool
	Value    string
	Filters  []FilterState
}

type FilterHandler func(result FilterResult)

type ThemeColors struct {
	Background    tcell.Color
	Text          tcell.Color
	HeaderText    tcell.Color
	ShortcutKey   tcell.Color
	Border        tcell.Color
	Selection     tcell.Color
	SelectionText tcell.Color
}

func DefaultTheme() ThemeColors {
	return ThemeColors{
		Background:    tcell.NewHexColor(0x000000), // Black background
		Text:          tcell.ColorWhite,            // White text
		HeaderText:    tcell.ColorYellow,           // Yellow headers
		ShortcutKey:   tcell.NewHexColor(0x017be9), // Blue shortcut keys
		Border:        tcell.NewHexColor(0x63a0bf), // Light blue border
		Selection:     tcell.NewHexColor(0x373737), // Dark grey for selection
		SelectionText: tcell.NewHexColor(0x00bebe), // Cyan for selected text
	}
}

func StyleText(text string, color tcell.Color) string {
	if color == tcell.ColorDefault {
		return text
	}
	return "[" + color.String() + "]" + text + "[-:-:-]"
}

func CreateFooter(theme ThemeColors) *tview.TextView {
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetText(StyleText("↑/↓", theme.ShortcutKey) +
			StyleText(" Navigate  ", tcell.ColorGray) +
			StyleText("Space", theme.ShortcutKey) +
			StyleText(" Select  ", tcell.ColorGray) +
			StyleText("Enter", theme.ShortcutKey) +
			StyleText(" Apply  ", tcell.ColorGray) +
			StyleText("Esc", theme.ShortcutKey) +
			StyleText(" Cancel", tcell.ColorGray)).
		SetTextAlign(tview.AlignCenter)
	footer.SetBackgroundColor(theme.Background)
	return footer
}

func CreateBoldFooter(theme ThemeColors) *tview.TextView {
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetText(StyleText("↑/↓", theme.ShortcutKey) +
			StyleText(" Navigate  ", tcell.ColorGray) +
			StyleText("Space", theme.ShortcutKey) +
			StyleText(" Toggle  ", tcell.ColorGray) +
			StyleText("Enter", theme.ShortcutKey) +
			StyleText(" Apply", tcell.ColorGray)).
		SetTextAlign(tview.AlignCenter)
	footer.SetBackgroundColor(theme.Background)
	return footer
}

func FindFilterByType(filters []FilterState, filterType FilterType) (string, bool) {
	for _, filter := range filters {
		if filter.Type == filterType {
			return filter.Value, true
		}
	}
	return "", false
}

func UpdateFilter(filters []FilterState, filterType FilterType, value string) []FilterState {
	if value == "" {
		result := make([]FilterState, 0, len(filters))
		for _, filter := range filters {
			if filter.Type != filterType {
				result = append(result, filter)
			}
		}
		return result
	}

	for i, filter := range filters {
		if filter.Type == filterType {
			filters[i].Value = value
			return filters
		}
	}

	return append(filters, FilterState{Type: filterType, Value: value})
}
