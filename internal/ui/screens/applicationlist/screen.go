package applicationlist

import (
	"fmt"
	"strings"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/Jack200062/ArguTUI/internal/ui/screens/applicationResourcesList"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ScreenAppList struct {
	app          *tview.Application
	instanceInfo *common.InstanceInfo
	apps         []argocd.Application
	client       *argocd.ArgoCdClient
	router       *ui.Router

	grid         *tview.Grid
	table        *tview.Table
	pages        *tview.Pages
	searchBar    *components.SearchBar
	filteredApps []argocd.Application
}

func New(
	app *tview.Application,
	c *argocd.ArgoCdClient,
	r *ui.Router,
	instanceInfo *common.InstanceInfo,
	apps []argocd.Application,
) *ScreenAppList {
	return &ScreenAppList{
		app:          app,
		client:       c,
		router:       r,
		instanceInfo: instanceInfo,
		apps:         apps,
		filteredApps: apps,
	}
}

func (s *ScreenAppList) Init() tview.Primitive {
	// Dedicated shortcut view in different package
	shortCutInfo := components.ShortcutBar()

	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String()).
		SetTextAlign(tview.AlignLeft)
	instanceBox.SetBorder(false)
	instanceBox.SetScrollable(true)

	topBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceBox, 0, 1, false).
		AddItem(shortCutInfo, 0, 1, false)

	s.searchBar = components.NewSearchBar("=> ", 0)
	s.searchBar.Filter = func(query string) []interface{} {
		var items []interface{}
		for _, app := range s.apps {
			items = append(items, app)
		}
		indices := components.FuzzyFilter(query, items)
		filtered := make([]interface{}, len(indices))
		for i, idx := range indices {
			filtered[i] = s.apps[idx]
		}
		return filtered
	}
	s.searchBar.OnDone = func(filtered []interface{}) {
		if filtered == nil {
			s.filteredApps = s.apps
		} else {
			s.filteredApps = make([]argocd.Application, len(filtered))
			for i, item := range filtered {
				s.filteredApps[i] = item.(argocd.Application)
			}
		}
		s.fillTable(s.filteredApps)
		s.hideSearchBar()
		s.app.SetFocus(s.table)
	}

	s.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
		// Set title without borders
	s.table.SetBorder(true).SetTitle(" ArgoCD Applications ")

	s.fillTable(s.filteredApps)

	s.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(0).
		SetBorders(true)

	s.grid.AddItem(topBar, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 0, 1, 1, 0, 0, true)

	s.pages = tview.NewPages().
		AddPage("main", s.grid, true, true)

	helpView := components.NewHelpView()
	helpView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			s.pages.HidePage("help")
			s.app.SetFocus(s.grid)
			return nil
		}
		return event
	})

	s.pages.AddPage("help", helpView, true, false)

	s.grid.SetInputCapture(s.onGridKey)

	return s.pages
}

func (s *ScreenAppList) onGridKey(event *tcell.EventKey) *tcell.EventKey {
	if s.app.GetFocus() == s.searchBar.InputField {
		return event
	}
	switch event.Rune() {
	case '?':
		s.pages.ShowPage("help")
		s.app.SetFocus(s.pages)
		return nil
	case 'q':
		s.app.Stop()
		return nil
	case '/', ':':
		s.showSearchBar()
		s.app.SetFocus(s.searchBar.InputField)
		return nil
	case 'd':
		// Обработка для 'd', если нужно
	}
	if event.Key() == tcell.KeyEnter {
		// При Enter выбираем приложение
		row, _ := s.table.GetSelection()
		if row < 1 || row-1 >= len(s.filteredApps) {
			return event
		}
		selectedApp := s.filteredApps[row-1]
		resources, err := s.client.GetAppResources(selectedApp.Name)
		if err != nil {
			modal := components.ErrorModal(
				fmt.Sprintf("Error getting resources for app %s:", selectedApp.Name),
				err.Error(),
				func() {
					s.app.SetRoot(s.grid, true)
				},
			)
			s.app.SetRoot(modal, true)
			return nil
		}
		resScreen := applicationResourcesList.New(s.app, resources, selectedApp.Name, s.router, s.instanceInfo)
		s.router.AddScreen(resScreen)
		s.router.SwitchTo(resScreen.Name())
		return nil
	}
	if event.Key() == tcell.KeyTAB {
		if s.table.HasFocus() {
			s.app.SetFocus(s.searchBar.InputField)
		} else {
			s.app.SetFocus(s.table)
		}
		return nil
	}
	return event
}

func (s *ScreenAppList) showSearchBar() {
	s.filteredApps = s.apps
	s.fillTable(s.apps)
	s.searchBar.InputField.SetText("")
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, 1, -1)
	s.grid.AddItem(s.searchBar.InputField, 1, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 2, 0, 1, 1, 0, 0, true)
	s.app.SetFocus(s.searchBar.InputField)
}

func (s *ScreenAppList) hideSearchBar() {
	s.grid.RemoveItem(s.searchBar.InputField)
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, -1)
	s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)
	s.searchBar.InputField.SetText("")
	s.app.SetFocus(s.table)
}

func (s *ScreenAppList) fillTable(apps []argocd.Application) {
	s.table.Clear()

	headers := []string{"Name", "Status", "Project"}
	for col, h := range headers {
		headerCell := tview.NewTableCell(fmt.Sprintf("[::b]%s", h)).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, headerCell)
	}

	row := 1
	for _, app := range apps {
		nameCell := tview.NewTableCell(app.Name).SetExpansion(1)
		statusCell := tview.NewTableCell(app.Status).SetExpansion(1)
		projectCell := tview.NewTableCell(app.Project).SetExpansion(1)

		switch strings.ToLower(app.Status) {
		case "healthy":
			statusCell.SetTextColor(tcell.ColorGreen)
		case "progressing":
			statusCell.SetTextColor(tcell.ColorOrange)
		case "suspended":
			statusCell.SetTextColor(tcell.ColorBlue)
		case "missing":
			statusCell.SetTextColor(tcell.ColorGrey)
		case "degraded":
			statusCell.SetTextColor(tcell.ColorRed)
		}

		s.table.SetCell(row, 0, nameCell)
		s.table.SetCell(row, 1, statusCell)
		s.table.SetCell(row, 2, projectCell)
		row++
	}
}

func (s *ScreenAppList) Name() string {
	return "ApplicationList"
}

var _ ui.Screen = (*ScreenAppList)(nil)
