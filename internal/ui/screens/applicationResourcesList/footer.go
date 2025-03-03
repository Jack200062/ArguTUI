package applicationResourcesList

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Footer struct {
	view             *tview.Flex
	infoView         *tview.TextView
	backgroundColor  tcell.Color
	shortcutKeyColor tcell.Color
	app              *tview.Application
	resourceCount    int
	ticker           *time.Ticker
	done             chan bool
}

func NewFooter(app *tview.Application, backgroundColor, shortcutKeyColor tcell.Color) *Footer {
	return &Footer{
		app:              app,
		backgroundColor:  backgroundColor,
		shortcutKeyColor: shortcutKeyColor,
		done:             make(chan bool),
	}
}

func (f *Footer) Init() tview.Primitive {
	footer := tview.NewFlex().
		SetDirection(tview.FlexColumn)
	footer.SetBackgroundColor(f.backgroundColor)

	leftPadding := tview.NewTextView()
	leftPadding.SetBackgroundColor(f.backgroundColor)
	footer.AddItem(leftPadding, 0, 1, false)

	commonShortcuts := map[string]string{
		"Enter": "Toggle expand",
		"b":     "Back",
		"q":     "Quit",
	}
	shortcutsView := components.NewHorizontalShortcutBar(
		commonShortcuts,
		f.backgroundColor,
		f.shortcutKeyColor,
	)
	footer.AddItem(shortcutsView, 0, 2, false)

	infoView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	infoView.SetBackgroundColor(f.backgroundColor)
	f.infoView = infoView
	footer.AddItem(infoView, 0, 1, false)

	f.view = footer

	f.UpdateResourceCount(0)

	return footer
}

func (f *Footer) Stop() {
	if f.ticker != nil {
		f.done <- true
	}
}

func (f *Footer) UpdateResourceCount(count int) {
	if f.infoView == nil {
		return
	}

	f.resourceCount = count

	now := time.Now()
	timeStr := now.Format("2nd January 15:04")

	day := now.Day()
	var suffix string
	switch day {
	case 1, 21, 31:
		suffix = "st"
	case 2, 22:
		suffix = "nd"
	case 3, 23:
		suffix = "rd"
	default:
		suffix = "th"
	}
	timeStr = strings.Replace(timeStr, "nd", suffix, 1)

	f.infoView.SetText(fmt.Sprintf("[gray]Total resources: [#63a0bf]%d[gray] | [#ffffff]%s",
		count, timeStr))
}

func (f *Footer) GetView() tview.Primitive {
	return f.view
}
