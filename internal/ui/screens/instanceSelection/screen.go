package screens

import (
	"fmt"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type InstanceSelectionScreen struct {
	app      *tview.Application
	cfg      *config.Config
	router   *ui.Router
	onSelect func(*config.Instance)

	list       *tview.List
	listWidth  int
	listHeight int
}

func NewInstanceSelectionScreen(
	app *tview.Application,
	cfg *config.Config,
	router *ui.Router,
	onSelect func(*config.Instance),
) *InstanceSelectionScreen {
	s := &InstanceSelectionScreen{
		app:      app,
		cfg:      cfg,
		router:   router,
		onSelect: onSelect,
		list:     tview.NewList(),
	}
	s.initList()
	s.calculateDimensions()
	return s
}

func (s *InstanceSelectionScreen) initList() {
	textColor := tcell.NewHexColor(0x00bebe)        // #795200 for text
	mainTextColor := tcell.NewHexColor(0x805700)    // #795200 for main text
	backgroundColor := tcell.NewHexColor(0x000000)  // #000000 for background
	borderColor := tcell.NewHexColor(0x63a0bf)      // #63a0bf for borders
	shortcutKeyColor := tcell.NewHexColor(0x017be9) // #003e77 for shortcut keys
	selectedBgColor := tcell.NewHexColor(0x373737)  // Darker variant for selection

	// Initialize the list with custom colors
	s.list.ShowSecondaryText(true).
		SetMainTextColor(mainTextColor).
		SetShortcutColor(shortcutKeyColor).
		SetSelectedBackgroundColor(selectedBgColor).
		SetSelectedTextColor(textColor).
		SetBackgroundColor(backgroundColor).
		SetBorderColor(borderColor).
		SetBorder(true).
		SetTitle(fmt.Sprintf(" ArgoCD Instances (%d) ", len(s.cfg.Instances))).
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(textColor)

	for i, inst := range s.cfg.Instances {
		localInst := inst
		shortcut := rune('1' + i)
		if i >= 9 {
			shortcut = rune('a' + i - 9)
		}

		mainText := fmt.Sprintf("(%s) %s", localInst.Name, localInst.Url)

		s.list.AddItem(mainText, "", shortcut, func() {
			s.onSelect(localInst)
		})
	}

	s.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			s.app.Stop()
			return nil
		}
		return event
	})
}

func (s *InstanceSelectionScreen) calculateDimensions() {
	numItems := len(s.cfg.Instances)
	if numItems == 0 {
		s.listHeight = 20
		s.listWidth = 40
		return
	}

	// Slightly larger height calculation
	s.listHeight = numItems*2 + 10 // More padding

	// Calculate width based on content
	maxMainLen := 0
	for _, inst := range s.cfg.Instances {
		if len(inst.Name) > maxMainLen {
			maxMainLen = len(inst.Name)
		}
	}

	// Use the longer of the two for width calculation
	contentWidth := maxMainLen + 10

	// Add more padding for borders, shortcut, etc.
	s.listWidth = contentWidth + 12

	// Ensure minimum reasonable width - larger than before
	if s.listWidth < 45 {
		s.listWidth = 45
	}
}

func (s *InstanceSelectionScreen) Init() tview.Primitive {
	// Colors
	textColor := tcell.ColorYellow                 // #795200
	backgroundColor := tcell.NewHexColor(0x000000) // #000000 for background
	// Create header with app title
	header := tview.NewTextView().
		SetTextColor(textColor).
		SetTextAlign(tview.AlignCenter).SetText(" ArguTUI - ArgoCD Terminal UI ")
	header.SetBackgroundColor(backgroundColor)

	// Create footer with keyboard shortcuts
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[#017be9]↑↓[gray] Navigate  [#017be9]Enter[gray] Select  [#017be9]Esc[gray] Back").
		SetTextAlign(tview.AlignCenter)
	footer.SetBackgroundColor(backgroundColor)

	// Combine in a flex layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(s.list, s.listHeight, 0, true).
		AddItem(footer, 1, 0, false)

	flex.SetBackgroundColor(backgroundColor)

	// Center the flex on screen
	centeredFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(flex, s.listWidth, 0, true).
				AddItem(nil, 0, 1, false),
			s.listHeight+2, 0, true).
		AddItem(nil, 0, 1, false)

	centeredFlex.SetBackgroundColor(backgroundColor)

	return centeredFlex
}

func (s *InstanceSelectionScreen) Name() string {
	return "InstanceSelection"
}
