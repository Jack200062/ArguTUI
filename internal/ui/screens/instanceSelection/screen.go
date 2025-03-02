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
	s.list.ShowSecondaryText(false).
		SetBorder(true).
		SetTitle(" Choose instance ").
		SetTitleAlign(tview.AlignCenter)

	for i, inst := range s.cfg.Instances {
		localInst := inst
		displayText := fmt.Sprintf("%d. %s (%s)", i+1, localInst.Name, localInst.Url)
		s.list.AddItem(displayText, "", 0, func() {
			s.onSelect(localInst)
		})
	}

	s.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			s.router.Back()
			return nil
		}
		return event
	})
}

func (s *InstanceSelectionScreen) calculateDimensions() {
	numItems := len(s.cfg.Instances)
	if numItems == 0 {
		s.listHeight = 5
		s.listWidth = 30
		return
	}
	s.listHeight = numItems*2 - 1 + 2

	maxLen := 0
	for i, inst := range s.cfg.Instances {
		str := fmt.Sprintf("%d. %s (%s)", i+1, inst.Name, inst.Url)
		if l := len(str); l > maxLen {
			maxLen = l
		}
	}
	s.listWidth = maxLen + 4
}

func (s *InstanceSelectionScreen) Init() tview.Primitive {
	s.list.SetRect(0, 0, s.listWidth, s.listHeight)

	verticalFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	verticalFlex.AddItem(nil, 0, 1, false)
	verticalFlex.AddItem(s.list, s.listHeight, 0, true)
	verticalFlex.AddItem(nil, 0, 1, false)

	horizontalFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	horizontalFlex.AddItem(nil, 0, 1, false)
	horizontalFlex.AddItem(verticalFlex, s.listWidth, 0, true)
	horizontalFlex.AddItem(nil, 0, 1, false)

	return horizontalFlex
}

func (s *InstanceSelectionScreen) Name() string {
	return "InstanceSelection"
}
