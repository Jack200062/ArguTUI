package applicationlist

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/Jack200062/ArguTUI/internal/ui/screens/applicationResourcesList"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func filterApplications(apps []argocd.Application, query string) []argocd.Application {
	var filtered []argocd.Application
	lowerQuery := strings.ToLower(query)
	for _, app := range apps {
		if strings.Contains(app.SearchString(), lowerQuery) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

type ScreenAppList struct {
	app          *tview.Application
	instanceInfo *common.InstanceInfo
	apps         []argocd.Application
	client       *argocd.ArgoCdClient
	router       *ui.Router

	grid         *tview.Grid
	table        *tview.Table
	pages        *tview.Pages
	searchBar    *components.SimpleSearchBar
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
	s.startAutoRefresh()

	shortCutInfo := components.ShortcutBar()

	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String()).
		SetTextAlign(tview.AlignLeft)
	instanceBox.SetBorder(false)

	topBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceBox, 0, 1, false).
		AddItem(shortCutInfo, 0, 1, false)

	s.searchBar = components.NewSimpleSearchBar("=> ", 20)
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

	s.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
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
	helpView.SetInputCapture(s.helpInputCapture)
	s.pages.AddPage("help", helpView, true, false)

	s.grid.SetInputCapture(s.onGridKey)

	return s.pages
}

func (s *ScreenAppList) searchDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		query := s.searchBar.InputField.GetText()
		s.filteredApps = filterApplications(s.apps, query)
		s.fillTable(s.filteredApps)
		s.hideSearchBar()
		s.app.SetFocus(s.table)
	}
}

func (s *ScreenAppList) helpInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Rune() == 'q' {
		s.pages.HidePage("help")
		s.app.SetFocus(s.grid)
		return nil
	}
	return event
}

func (s *ScreenAppList) startAutoRefresh() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			s.app.QueueUpdateDraw(func() {
				newApps, err := s.client.GetApps()
				if err != nil {
					return
				}
				s.apps = newApps
				s.filteredApps = newApps
				s.fillTable(s.filteredApps)
			})
		}
	}()
}

func (s *ScreenAppList) refreshApps() {
	newApps, err := s.client.GetApps()
	if err != nil {
		return
	}
	s.apps = newApps
	s.filteredApps = newApps
	s.fillTable(s.filteredApps)
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
	case 'R':
		s.refreshApps()
		return nil
	case 'r':
		row, _ := s.table.GetSelection()
		if row < 1 || row-1 >= len(s.filteredApps) {
			return event
		}
		selectedApp := s.filteredApps[row-1]
		err := s.client.RefreshApp(selectedApp.Name, "normal")
		if err != nil {
			modal := components.ErrorModal(
				fmt.Sprintf("Error refreshing app %s:", selectedApp.Name),
				err.Error(),
				s.modalClose,
			)
			s.app.SetRoot(modal, true)
			return nil
		}
		s.showToast(fmt.Sprintf("App %s refreshed successfully!", selectedApp.Name), 2*time.Second)
		return nil
	}
	if event.Key() == tcell.KeyEnter {
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
				s.modalClose,
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

func (s *ScreenAppList) modalClose() {
	s.app.SetRoot(s.pages, true)
}

func (s *ScreenAppList) showToast(message string, duration time.Duration) {
	var toast *components.SimpleSearchBar
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, 1, -1)
	toast = components.NewSimpleSearchBar("✅  ", 0)
	toast.InputField.SetText(message)
	s.grid.AddItem(toast.InputField, 1, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 2, 0, 1, 1, 0, 0, true)
	go func() {
		time.Sleep(duration)
		s.app.QueueUpdateDraw(func() {
			s.grid.RemoveItem(toast.InputField)
			s.grid.RemoveItem(s.table)
			s.grid.SetRows(3, 0)
			s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)
		})
	}()
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
			nameCell.SetTextColor(tcell.ColorGreen)
			statusCell.SetTextColor(tcell.ColorGreen)
			projectCell.SetTextColor(tcell.ColorGreen)
		case "progressing":
			nameCell.SetTextColor(tcell.ColorOrange)
			statusCell.SetTextColor(tcell.ColorOrange)
			projectCell.SetTextColor(tcell.ColorOrange)
		case "suspended":
			nameCell.SetTextColor(tcell.ColorBlue)
			statusCell.SetTextColor(tcell.ColorBlue)
			projectCell.SetTextColor(tcell.ColorBlue)
		case "missing":
			nameCell.SetTextColor(tcell.ColorRed)
			statusCell.SetTextColor(tcell.ColorGrey)
			projectCell.SetTextColor(tcell.ColorRed)
		case "degraded":
			nameCell.SetTextColor(tcell.ColorRed)
			statusCell.SetTextColor(tcell.ColorRed)
			projectCell.SetTextColor(tcell.ColorRed)
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
