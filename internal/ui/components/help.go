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

// HelpSection представляет секцию справки
type HelpSection struct {
	Title     string
	Shortcuts map[string]string
}

// NewHelpView создает новое представление справки с гибкой конфигурацией
func NewHelpView(options ...func(*HelpView)) *HelpView {
	// Цвета по умолчанию
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

	// Применяем ползовательские опции
	for _, option := range options {
		option(helpView)
	}

	helpView.createView()
	return helpView
}

// WithBackgroundColor устанавливает цвет фона
func WithBackgroundColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.backgroundColor = color
	}
}

// WithTextColor устанавливает цвет текста
func WithTextColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.textColor = color
	}
}

// WithHeaderColor устанавливает цвет заголовков
func WithHeaderColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.headerColor = color
	}
}

// WithKeyColor устанавливает цвет клавиш
func WithKeyColor(color tcell.Color) func(*HelpView) {
	return func(h *HelpView) {
		h.keyColor = color
	}
}

// createView создает TextView для отображения справки
func (h *HelpView) createView() {
	h.View = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetWrap(true)

	h.View.SetBackgroundColor(h.backgroundColor)
	h.View.SetTextColor(h.textColor)
}

// GenerateSections возвращает стандартные секции справки
func (h *HelpView) GenerateSections() []HelpSection {
	return []HelpSection{
		{
			Title: "GENERAL",
			Shortcuts: map[string]string{
				"q": "Close help screen",
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
				"p, P": "Toggle Progressing filter",
				"s":    "Toggle Synced filter",
				"o, O": "Toggle OutOfSync filter",
				"c, C": "Clear all filters",
			},
		},
		{
			Title: "RESOURCES",
			Shortcuts: map[string]string{
				"t":   "Toggle resource tree expansion",
				"d":   "Filter by Deployments",
				"s":   "Filter by Services",
				"i":   "Filter by Ingress",
				"c":   "Filter by ConfigMaps",
				"Tab": "Navigate to next section",
			},
		},
	}
}

// RenderHelp генерирует текст справки с цветным форматированием
func (h *HelpView) RenderHelp() string {
	var helpText strings.Builder

	helpText.WriteString(fmt.Sprintf("[%s::b]ArguTUI - KEYBOARD SHORTCUTS[-:-:-]\n\n", h.headerColor))

	for _, section := range h.GenerateSections() {
		// Форматируем заголовок секции
		helpText.WriteString(fmt.Sprintf("[%s::b]%s:[-:-:-]\n", h.headerColor, section.Title))

		// Форматируем каждый шорткат
		for key, desc := range section.Shortcuts {
			helpText.WriteString(fmt.Sprintf("[%s]%s[%s] %s\n",
				h.keyColor.String(),
				key,
				h.textColor.String(),
				desc))
		}
		helpText.WriteString("\n")
	}

	return helpText.String()
}

// Render возвращает готовый компонент TextView
func (h *HelpView) Render() *tview.TextView {
	h.View.SetText(h.RenderHelp())
	return h.View
}

// GetInputCapture возвращает обработчик ввода для закрытия справки
func (h *HelpView) GetInputCapture(onClose func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			if onClose != nil {
				onClose()
			}
			return nil
		}
		return event
	}
}
