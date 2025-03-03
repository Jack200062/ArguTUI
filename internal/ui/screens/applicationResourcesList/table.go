package applicationResourcesList

import (
	"fmt"
	"strings"

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
	appName         string
}

func NewTableView(appName string, textColor, borderColor, backgroundColor, selectedBgColor tcell.Color) *TableView {
	return &TableView{
		appName:         appName,
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
		SetTitle(fmt.Sprintf(" Resources for %s ", t.appName)).
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(t.textColor).
		SetBorderColor(t.borderColor).
		SetBackgroundColor(t.backgroundColor)

	return t.table
}

func (t *TableView) FillTableWithTree(resources []*TreeResource, activeFilters string) {
	t.table.Clear()
	headers := []string{"Kind", "Name", "Health", "SyncStatus", "Namespace"}
	for col, h := range headers {
		headerCell := tview.NewTableCell(fmt.Sprintf("[::b]%s", h)).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		t.table.SetCell(0, col, headerCell)
	}

	title := fmt.Sprintf(" Resources for %s ", t.appName)
	if activeFilters != "" {
		title = fmt.Sprintf(" Resources for %s (%s) ", t.appName, activeFilters)
	}
	t.table.SetTitle(title)

	row := 1
	for _, tr := range resources {
		var lineInfo *TreeLineInfo

		treePrefix := generateTreePrefix(tr, lineInfo)
		kindText := treePrefix + tr.Kind

		kindCell := tview.NewTableCell(kindText).SetExpansion(1)

		t.table.SetCell(row, 0, kindCell)
		t.table.SetCell(row, 1, tview.NewTableCell(tr.Name).SetExpansion(1))

		healthCell := tview.NewTableCell(tr.Health).
			SetExpansion(1).
			SetTextColor(common.ColorForHealthStatus(tr.Health))
		t.table.SetCell(row, 2, healthCell)

		syncCell := tview.NewTableCell(tr.SyncStatus).
			SetExpansion(1)
		if tr.SyncStatus == "OutOfSync" {
			syncCell.SetTextColor(tcell.ColorYellow)
		} else {
			syncCell.SetTextColor(tcell.ColorGreen)
		}
		t.table.SetCell(row, 3, syncCell)

		t.table.SetCell(row, 4, tview.NewTableCell(tr.Namespace).SetExpansion(1))

		rowColor := common.RowColorForStatuses(tr.Health, tr.SyncStatus)
		common.SetRowColor(t.table, row, len(headers), rowColor)

		row++
	}
}

func (t *TableView) FillTableWithSearch(resources []*TreeResource) {
	t.table.Clear()
	headers := []string{"Kind", "Name", "Health", "SyncStatus", "Namespace"}
	for col, h := range headers {
		cell := tview.NewTableCell("[::b]" + h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		t.table.SetCell(0, col, cell)
	}

	t.table.SetTitle(fmt.Sprintf(" Search Results for %s ", t.appName))

	row := 1
	for _, tr := range resources {
		kindCell := tview.NewTableCell(tr.Kind).SetExpansion(1)

		t.table.SetCell(row, 0, kindCell)
		t.table.SetCell(row, 1, tview.NewTableCell(tr.Name).SetExpansion(1))

		healthCell := tview.NewTableCell(tr.Health).
			SetExpansion(1).
			SetTextColor(common.ColorForHealthStatus(tr.Health))
		t.table.SetCell(row, 2, healthCell)

		syncCell := tview.NewTableCell(tr.SyncStatus).
			SetExpansion(1)
		if tr.SyncStatus == "OutOfSync" {
			syncCell.SetTextColor(tcell.ColorYellow)
		} else {
			syncCell.SetTextColor(tcell.ColorGreen)
		}
		t.table.SetCell(row, 3, syncCell)

		t.table.SetCell(row, 4, tview.NewTableCell(tr.Namespace).SetExpansion(1))

		rowColor := common.RowColorForStatuses(tr.Health, tr.SyncStatus)
		common.SetRowColor(t.table, row, len(headers), rowColor)

		row++
	}
}

func (t *TableView) GetTable() *tview.Table {
	return t.table
}

func generateTreePrefix(node *TreeResource, parentLineInfo *TreeLineInfo) string {
	if node.Depth == 0 {
		if len(node.Children) > 0 {
			if node.Expanded {
				return "▼ "
			}
			return "▶ "
		}
		return "  "
	}

	var prefix strings.Builder

	for i := 0; i < node.Depth-1; i++ {
		if parentLineInfo != nil && i < len(parentLineInfo.LineChars) {
			if parentLineInfo.LineChars[i] == '│' {
				prefix.WriteString("│ ")
			} else {
				prefix.WriteString("  ")
			}
		} else {
			prefix.WriteString("  ")
		}
	}

	if node.IsLast {
		prefix.WriteString("└─")
	} else {
		prefix.WriteString("├─")
	}

	if len(node.Children) > 0 {
		if node.Expanded {
			prefix.WriteString("▼ ")
		} else {
			prefix.WriteString("▶ ")
		}
	} else {
		prefix.WriteString("  ")
	}

	return prefix.String()
}
