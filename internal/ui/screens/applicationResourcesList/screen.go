package applicationResourcesList

import (
	"fmt"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ScreenAppResourcesList struct {
	app       *tview.Application
	appName   string
	resources []argocd.Resource
	router    *ui.Router

	table *tview.Table
	grid  *tview.Grid
}

func New(app *tview.Application, resources []argocd.Resource, appName string, r *ui.Router) *ScreenAppResourcesList {
	return &ScreenAppResourcesList{
		app:       app,
		appName:   appName,
		resources: resources,
		router:    r,
	}
}

func (s *ScreenAppResourcesList) Init() tview.Primitive {
	// 1. Top bar with shortcuts
	topBar := tview.NewTextView().
		SetText(" <TAB> Switch Panel   q Quit   d Details ").
		SetTextAlign(tview.AlignLeft)

	// 3. Initialize the table (without chaining .SetBorder)
	resourcesBoxTitle := fmt.Sprintf("Resources %s", s.appName)
	s.table = tview.NewTable().
		SetSelectable(true, false).
		SetBorders(false)
	s.table.Box.SetBorder(true)
	s.table.Box.SetTitle(resourcesBoxTitle) // Select rows, not columns

	headers := []string{"Kind", "Name", "Namespace"}
	for col, header := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", header)).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, cell)
	}

	// Fill data rows, also using .SetExpansion(1) on each cell
	row := 1
	for _, res := range s.resources {
		s.table.SetCell(row, 0, tview.NewTableCell(res.Kind).SetExpansion(1))
		s.table.SetCell(row, 1, tview.NewTableCell(res.Name).SetExpansion(1))
		s.table.SetCell(row, 2, tview.NewTableCell(res.Namespace).SetExpansion(1))
		row++
	}

	// 5. Use a grid to place topBar, instanceBox (left) and table (right/below)
	s.grid = tview.NewGrid().
		SetRows(3, 0). // 3 строки для верхней панели (можно регулировать) и оставшееся пространство для таблицы
		SetColumns(0).
		SetBorders(true)

	s.grid.AddItem(topBar, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 0, 1, 1, 0, 0, true)

		// 6. Global key handling
	s.grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'b':
			s.router.Back()
			return nil
		case 'q':
			s.app.Stop()
			return nil
		}
		return event
	})

	return s.grid
}

func (s *ScreenAppResourcesList) Name() string {
	return "ApplicationResourcesList"
}

var _ ui.Screen = (*ScreenAppResourcesList)(nil)
