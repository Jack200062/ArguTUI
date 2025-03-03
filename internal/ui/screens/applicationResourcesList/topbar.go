package applicationResourcesList

import (
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TopBar struct {
	view             *tview.Flex
	instanceInfo     *common.InstanceInfo
	infoView         *tview.TextView
	backgroundColor  tcell.Color
	shortcutKeyColor tcell.Color
	textColor        tcell.Color
	appName          string
	kindFilter       string
}

func NewTopBar(
	instanceInfo *common.InstanceInfo,
	appName string,
	backgroundColor,
	shortcutKeyColor,
	textColor tcell.Color,
) *TopBar {
	return &TopBar{
		instanceInfo:     instanceInfo,
		appName:          appName,
		backgroundColor:  backgroundColor,
		shortcutKeyColor: shortcutKeyColor,
		textColor:        textColor,
	}
}

func (t *TopBar) Init() tview.Primitive {
	instanceView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(t.instanceInfo.FormattedString(tcell.ColorYellow)).
		SetTextAlign(tview.AlignLeft)
	instanceView.SetBorder(false)
	instanceView.SetBackgroundColor(t.backgroundColor)
	t.infoView = instanceView

	shortcutBar := components.NewShortcutBar(t.backgroundColor, t.shortcutKeyColor)

	shortcutBar.AddGroup("Resources", map[string]string{
		"f":     "Filter",
		"t":     "Toggle Tree",
		"d/s/i": "Filter Kind",
		"/":     "Search",
	})

	shortcutBar.AddGroup("Navigation", map[string]string{
		"b": "Back",
		"q": "Quit",
	})

	shortcutBarPrimitive := shortcutBar.Init()

	t.view = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceView, 0, 1, false).
		AddItem(shortcutBarPrimitive, 0, 2, false)
	t.view.SetBackgroundColor(t.backgroundColor)

	return t.view
}

func (t *TopBar) UpdateFilter(kindFilter string) {
	t.kindFilter = kindFilter
}

func (t *TopBar) GetView() tview.Primitive {
	return t.view
}
