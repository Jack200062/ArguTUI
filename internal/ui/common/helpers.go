package common

import (
	"strings"

	ui "github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ColorForHealthStatus(status string) tcell.Color {
	switch strings.ToLower(status) {
	case "healthy":
		return tcell.ColorGreen
	case "progressing":
		return tcell.ColorOrange
	case "suspended":
		return tcell.ColorBlue
	case "missing":
		return tcell.ColorGrey
	case "degraded":
		return tcell.ColorRed
	default:
		return tcell.ColorWhite
	}
}

func RowColorForStatuses(healthStatus, syncStatus string) tcell.Color {
	if strings.ToLower(healthStatus) != "healthy" {
		return ColorForHealthStatus(healthStatus)
	} else if strings.ToLower(syncStatus) != "synced" {
		return tcell.ColorOrange
	}
	return ColorForHealthStatus("healthy")
}

func SetRowColor(table *tview.Table, row, columns int, color tcell.Color) {
	for col := 0; col < columns; col++ {
		if cell := table.GetCell(row, col); cell != nil {
			cell.SetTextColor(color)
		}
	}
}

func SetupExitHandler(tviewApp *tview.Application, router *ui.Router) {
	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if (event.Rune() == 'q' || event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyEsc) && !router.IsModalActive() {
			modal := tview.NewModal().
				SetText("Are you sure you want to close?").
				AddButtons([]string{"Yes", "No"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonIndex == 0 {
						tviewApp.Stop()
					} else {
						router.CloseModal()
					}
				}).SetButtonTextColor(tcell.NewHexColor(0x017be9))

			backgroundColor := tcell.NewHexColor(0x000000)
			textColor := tcell.NewHexColor(0x805700)

			modal.SetBackgroundColor(backgroundColor)
			modal.SetTextColor(textColor)
			modal.SetBorderColor(backgroundColor)

			router.ShowModal(modal)
			return nil
		}

		return event
	})
}
