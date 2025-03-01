package common

import (
	"strings"

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
