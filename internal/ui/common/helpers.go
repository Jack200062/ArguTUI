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

func SetRowColor(table *tview.Table, row, columns int, color tcell.Color) {
	for col := 0; col < columns; col++ {
		cell := table.GetCell(row, col)
		if cell != nil {
			cell.SetTextColor(color)
		}
	}
}
