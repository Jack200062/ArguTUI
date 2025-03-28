package applicationlist

import (
	"fmt"

	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TopBar struct {
	view             *tview.Flex
	instanceInfo     *common.InstanceInfo
	statsView        *tview.TextView
	instanceInfoView *tview.TextView
	backgroundColor  tcell.Color
	shortcutKeyColor tcell.Color
	textColor        tcell.Color
}

func NewTopBar(instanceInfo *common.InstanceInfo, backgroundColor, shortcutKeyColor, textColor tcell.Color) *TopBar {
	return &TopBar{
		instanceInfo:     instanceInfo,
		backgroundColor:  backgroundColor,
		shortcutKeyColor: shortcutKeyColor,
		textColor:        textColor,
	}
}

func (t *TopBar) Init() tview.Primitive {
	instanceInfoView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(t.instanceInfo.FormattedString(tcell.ColorYellow)).
		SetTextAlign(tview.AlignLeft)
	instanceInfoView.SetBorder(false)
	t.instanceInfoView = instanceInfoView

	statsView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	t.statsView = statsView

	shortcutBar := components.NewShortcutBar(t.backgroundColor, t.shortcutKeyColor)

	shortcutBar.AddGroup("App", map[string]string{
		"f":     "Filter",
		"h/d/p": "Sort By Health",
		"s/o":   "Sort By Sync",
	})

	shortcutBar.AddGroup("Actions", map[string]string{
		"R": "Refresh All",
		"r": "Refresh App",
		"c": "Clear Filters",
		"d": "Delete App",
	})

	shortcutBarPrimitive := shortcutBar.Init()

	t.view = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceInfoView, 0, 1, false).
		AddItem(statsView, 0, 1, false).
		AddItem(shortcutBarPrimitive, 0, 2, false)
	t.view.SetBackgroundColor(t.backgroundColor)

	return t.view
}

func (t *TopBar) UpdateStats(healthy, degraded, outOfSync int) {
	t.statsView.SetText(fmt.Sprintf("[green]Healthy: %d  [red]Degraded: %d  [yellow]OutOfSync: %d",
		healthy, degraded, outOfSync))
}
