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
	shortCutInfo := tview.NewTextView().
		SetText(" <TAB> Switch Panel   q Quit   d Details b Go back ").
		SetTextAlign(tview.AlignCenter)

	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String()).
		SetTextAlign(tview.AlignLeft)
	instanceBox.SetBorder(false)
	instanceBox.SetScrollable(true)

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	table.Box.SetBorder(true).
		SetTitle(" ArgoCD Applications ")
	s.table = table

	topBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceBox, 0, 1, false).
		AddItem(shortCutInfo, 0, 1, false)

	headers := []string{"Name", "Status", "Project"}
	for col, h := range headers {
		headerCell := tview.NewTableCell(fmt.Sprintf("[::b]%s", h)).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, headerCell)
	}

	row := 1
	for _, app := range s.apps {
		nameCell := tview.NewTableCell(app.Name).SetExpansion(1)
		statusCell := tview.NewTableCell(app.Status).SetExpansion(1)
		projectCell := tview.NewTableCell(app.Project).SetExpansion(1)

		if app.Status == "Healthy" {
			statusCell.SetTextColor(tcell.ColorGreen)
		} else if app.Status == "Progressing" {
			statusCell.SetTextColor(tcell.ColorOrange)
		} else if app.Status == "Suspended" {
			statusCell.SetTextColor(tcell.ColorBlue)
		} else if app.Status == "Missing" {
			statusCell.SetTextColor(tcell.ColorGrey)
		} else if app.Status == "Degraded" {
			statusCell.SetTextColor(tcell.ColorRed)
		}

		s.table.SetCell(row, 0, nameCell)
		s.table.SetCell(row, 1, statusCell)
		s.table.SetCell(row, 2, projectCell)
		row++
	}

	// 4. Новая конфигурация сетки
	s.grid = tview.NewGrid().
		SetRows(3, 0). // Строка 0: шорткаты (высота 1), Строка 1: instanceBox (высота 3), Строка 2: таблица (оставшееся)
		SetColumns(0). // Одна колонка на всю ширину
		SetBorders(true)

	s.grid.AddItem(topBar, 0, 0, 1, 1, 0, 0, false). // Шорткаты вврху
								AddItem(s.table, 1, 0, 1, 1, 0, 0, true) // Таблица внизу

	// 5. Обработка клавиш (без изменений)
	s.grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			s.app.Stop()
			return nil
		case 'd':
			// Дополнительная обработка для 'd', если нужно
		}
		if event.Key() == tcell.KeyEnter {
			row, _ := s.table.GetSelection()
			if row < 1 || row-1 >= len(s.apps) {
				return event
			}
			selectedApp := s.apps[row-1]
			resources, err := s.client.GetAppResources(selectedApp.Name)
			if err != nil {
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Error getting resources for app %s:\n\n%v", selectedApp.Name, err)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						s.app.SetRoot(s.grid, true)
					})
				s.app.SetRoot(modal, true)
				return nil
			}
			resScreen := applicationResourcesList.New(s.app, resources, selectedApp.Name, s.router)
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
