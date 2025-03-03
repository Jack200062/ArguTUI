package applicationlist

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
	timeView         *tview.TextView
	backgroundColor  tcell.Color
	shortcutKeyColor tcell.Color
	app              *tview.Application
	lastRefreshTime  time.Time
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
		"q": "Quit",
		"?": "Help",
		"b": "Back",
	}
	shortcutsView := components.NewHorizontalShortcutBar(
		commonShortcuts,
		f.backgroundColor,
		f.shortcutKeyColor,
	)
	footer.AddItem(shortcutsView, 0, 2, false)

	timeView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	timeView.SetBackgroundColor(f.backgroundColor)
	f.timeView = timeView
	footer.AddItem(timeView, 0, 1, false)

	f.view = footer

	f.startTimeUpdater()

	return footer
}

func (f *Footer) startTimeUpdater() {
	f.ticker = time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-f.ticker.C:
				f.app.QueueUpdateDraw(func() {
					f.UpdateTimeInfo(f.lastRefreshTime)
				})
			case <-f.done:
				f.ticker.Stop()
				return
			}
		}
	}()
}

func (f *Footer) Stop() {
	if f.ticker != nil {
		f.done <- true
	}
}

func (f *Footer) UpdateTimeInfo(lastRefreshTime time.Time) {
	if f.timeView == nil {
		return
	}

	f.lastRefreshTime = lastRefreshTime

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

	lastRefresh := f.getLastRefreshTime(lastRefreshTime)
	f.timeView.SetText(fmt.Sprintf("[gray]Last refresh: [#63a0bf]%s[gray] | [#ffffff]%s",
		lastRefresh, timeStr))
}

func (f *Footer) getLastRefreshTime(lastRefreshTime time.Time) string {
	elapsed := time.Since(lastRefreshTime)

	if elapsed < 10*time.Second {
		return "just now"
	} else if elapsed < time.Minute {
		return fmt.Sprintf("%d sec ago", int(elapsed.Seconds()))
	} else if elapsed < time.Hour {
		minutes := int(elapsed.Minutes())
		return fmt.Sprintf("%d min ago", minutes)
	} else if elapsed < 24*time.Hour {
		hours := int(elapsed.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	}
	return lastRefreshTime.Format("Jan 2 15:04")
}
