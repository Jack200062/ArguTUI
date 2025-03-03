package applicationlist

import (
	"fmt"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TableView struct {
	table           *tview.Table
	textColor       tcell.Color
	borderColor     tcell.Color
	backgroundColor tcell.Color
	selectedBgColor tcell.Color
}

func NewTableView(textColor, borderColor, backgroundColor, selectedBgColor tcell.Color) *TableView {
	return &TableView{
		textColor:       textColor,
		borderColor:     borderColor,
		backgroundColor: backgroundColor,
		selectedBgColor: selectedBgColor,
	}
}

func (t *TableView) Init() *tview.Table {
	t.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(t.selectedBgColor).
			Foreground(t.textColor))
	t.table.SetBorder(true).
		SetTitle(" Applications ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(t.textColor).
		SetBorderColor(t.borderColor).
		SetBackgroundColor(t.backgroundColor)

	return t.table
}

func (t *TableView) FillTable(apps []argocd.Application, activeFilters string) {
	t.table.Clear()
	headers := []string{"Name", "HealthStatus", "SyncStatus", "SyncCommit", "Project", "LastActivity"}
	for col, h := range headers {
		headerCell := tview.NewTableCell(fmt.Sprintf("[::b]%s", h)).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		t.table.SetCell(0, col, headerCell)
	}

	title := " Applications "
	if activeFilters != "" {
		title = fmt.Sprintf(" Applications (%s) ", activeFilters)
	}
	t.table.SetTitle(title)

	row := 1
	for _, app := range apps {
		nameCell := tview.NewTableCell(app.Name).SetExpansion(1)
		healthStatusCell := tview.NewTableCell(app.HealthStatus).SetExpansion(1)
		syncStatusCell := tview.NewTableCell(app.SyncStatus).SetExpansion(1)
		syncCommitCell := tview.NewTableCell(app.SyncCommit).SetExpansion(1)
		projectCell := tview.NewTableCell(app.Project).SetExpansion(1)
		lastActivityCell := tview.NewTableCell(app.LastActivity).SetExpansion(1)

		t.table.SetCell(row, 0, nameCell)
		t.table.SetCell(row, 1, healthStatusCell)
		t.table.SetCell(row, 2, syncStatusCell)
		t.table.SetCell(row, 3, syncCommitCell)
		t.table.SetCell(row, 4, projectCell)
		t.table.SetCell(row, 5, lastActivityCell)

		rowColor := common.RowColorForStatuses(app.HealthStatus, app.SyncStatus)
		common.SetRowColor(t.table, row, len(headers), rowColor)

		row++
	}
}

func (t *TableView) GetTable() *tview.Table {
	return t.table
}
