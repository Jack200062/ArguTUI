package applicationlist

import (
	"fmt"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/screens/applicationResourcesList"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ScreenAppList implements the ui.Screen interface for listing ArgoCD applications.
type ScreenAppList struct {
	app          *tview.Application
	instanceInfo *common.InstanceInfo
	apps         []argocd.Application
	client       *argocd.ArgoCdClient
	router       *ui.Router

	table *tview.Table
	grid  *tview.Grid
}

// New creates a new screen for listing ArgoCD applications.
func New(app *tview.Application, c *argocd.ArgoCdClient, r *ui.Router, instanceInfo *common.InstanceInfo, apps []argocd.Application) *ScreenAppList {
	return &ScreenAppList{
		app:          app,
		client:       c,
		router:       r,
		instanceInfo: instanceInfo,
		apps:         apps,
	}
}

func (s *ScreenAppList) Init() tview.Primitive {
	// 1. Top bar with shortcuts
	topBar := tview.NewTextView().
		SetText(" <TAB> Switch Panel   q Quit   d Details ").
		SetTextAlign(tview.AlignLeft)

	// 2. Box with ArgoCD instance info
	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String()).
		SetTextAlign(tview.AlignLeft)
	instanceBox.SetBorder(true).
		SetTitle(" ArgoCD Instance ")

	// 3. Initialize the table (without chaining .SetBorder)
	table := tview.NewTable().
		SetBorders(false).         // No cell borders inside the table
		SetSelectable(true, false) // Select rows, not columns

	// Now set border on the Box field:
	table.Box.SetBorder(true).
		SetTitle(" ArgoCD Applications ")
	s.table = table

	// 4. Suppose we have 3 columns: Name, Status, Project
	//    We'll expand them equally using TableCell.SetExpansion(1).
	headers := []string{"Name", "Status", "Project"}
	for col, h := range headers {
		headerCell := tview.NewTableCell(fmt.Sprintf("[::b]%s", h)).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1) // This cell expands horizontally
		s.table.SetCell(0, col, headerCell)
	}

	// Fill data rows, also using .SetExpansion(1) on each cell
	row := 1
	for _, app := range s.apps {
		nameCell := tview.NewTableCell(app.Name).SetExpansion(1)
		statusCell := tview.NewTableCell(app.Status).SetExpansion(1)
		projectCell := tview.NewTableCell(app.Project).SetExpansion(1)

		s.table.SetCell(row, 0, nameCell)
		s.table.SetCell(row, 1, statusCell)
		s.table.SetCell(row, 2, projectCell)
		row++
	}

	// 5. Use a grid to place topBar, instanceBox (left) and table (right/below)
	s.grid = tview.NewGrid().
		SetRows(1, 0).     // row0 = topBar, row1 = the rest
		SetColumns(30, 0). // col0 = instanceBox (width=30), col1 = table (expand)
		SetBorders(true)

	s.grid.AddItem(topBar, 0, 0, 1, 2, 0, 0, false).
		AddItem(instanceBox, 1, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 1, 1, 1, 0, 0, true)

	// 6. Global key handling
	s.grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			s.app.Stop()
			return nil
		case 'd':
			// Additional handling for 'd', if needed.
		}
		if event.Key() == tcell.KeyEnter {
			// Get the currently selected row.
			row, _ := s.table.GetSelection()
			if row < 1 || row-1 >= len(s.apps) {
				return event // No valid selection.
			}
			selectedApp := s.apps[row-1]
			// Fetch resources for the selected application.
			resources, err := s.client.GetAppResources(selectedApp.Name)
			if err != nil {
				// Создаем модальное окно ошибки. !!! Make more smooth
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Error getting resources for app %s:\n\n%v", selectedApp.Name, err)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						// После нажатия "OK" возвращаемся к текущему экрану.
						s.app.SetRoot(s.grid, true)
					})
				// Показываем модальное окно.
				s.app.SetRoot(modal, true)
				return nil
			}
			// Create the new screen for displaying resources.
			resScreen := applicationResourcesList.New(s.app, resources, selectedApp.Name, s.router)
			// Switch to the new screen.
			s.router.AddScreen(resScreen)
			s.router.SwitchTo(resScreen.Name())
			return nil
		}
		if event.Key() == tcell.KeyTAB {
			if s.table.HasFocus() {
				s.app.SetFocus(instanceBox)
			} else {
				s.app.SetFocus(s.table)
			}
			return nil
		}
		return event
	})

	return s.grid
}

func (s *ScreenAppList) Name() string {
	return "ApplicationList"
}

var _ ui.Screen = (*ScreenAppList)(nil)
