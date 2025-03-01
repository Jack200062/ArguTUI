package applicationResourcesList

import (
	"fmt"
	"strings"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func filterResources(resources []argocd.Resource, query string) []argocd.Resource {
	var filtered []argocd.Resource
	lowerQuery := strings.ToLower(query)
	for _, res := range resources {
		if strings.Contains(res.SearchString(), lowerQuery) {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

type ScreenAppResourcesList struct {
	app             *tview.Application
	appName         string
	resources       []argocd.Resource
	router          *ui.Router
	instanceInfo    *common.InstanceInfo
	table           *tview.Table
	grid            *tview.Grid
	pages           *tview.Pages
	searchBar       *components.SimpleSearchBar
	filteredResults []argocd.Resource
}

func New(app *tview.Application, resources []argocd.Resource, appName string, r *ui.Router, instanceInfo *common.InstanceInfo) *ScreenAppResourcesList {
	return &ScreenAppResourcesList{
		app:             app,
		appName:         appName,
		resources:       resources,
		filteredResults: resources,
		router:          r,
		instanceInfo:    instanceInfo,
	}
}

func (s *ScreenAppResourcesList) Init() tview.Primitive {
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

	resourcesBoxTitle := fmt.Sprintf(" Resources for %s ", s.appName)
	s.table = tview.NewTable().
		SetSelectable(true, false).
		SetBorders(false)
	s.table.Box.SetBorder(true)
	s.table.Box.SetTitle(resourcesBoxTitle)

	s.searchBar = components.NewSimpleSearchBar("=> ", 20)
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

	s.fillTable(s.filteredResults)

	s.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(0).
		SetBorders(true)
	s.grid.AddItem(topBar, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 0, 1, 1, 0, 0, true)

	s.pages = tview.NewPages().
		AddPage("main", s.grid, true, true)

	helpView := components.NewHelpView()
	helpView.SetInputCapture(s.helpKeyHandler)
	s.pages.AddPage("help", helpView, true, false)

	s.grid.SetInputCapture(s.globalKeyHandler)

	return s.pages
}

func (s *ScreenAppResourcesList) searchDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		query := s.searchBar.InputField.GetText()
		s.filteredResults = filterResources(s.resources, query)
		s.fillTable(s.filteredResults)
		s.hideSearchBar()
		s.app.SetFocus(s.table)
	}
}

func (s *ScreenAppResourcesList) helpKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	if event.Rune() == 'q' {
		s.pages.HidePage("help")
		s.app.SetFocus(s.grid)
		return nil
	}
	return event
}

func (s *ScreenAppResourcesList) globalKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	if s.app.GetFocus() == s.searchBar.InputField {
		return event
	}
	switch event.Rune() {
	case '?':
		s.pages.ShowPage("help")
		s.app.SetFocus(s.pages)
		return nil
	case 'I':
		s.router.SwitchTo("InstanceSelection")
		return nil
	case 'q':
		s.app.Stop()
		return nil
	case '/', ':':
		s.showSearchBar()
		s.app.SetFocus(s.searchBar.InputField)
		return nil
	case 'b':
		s.router.Back()
		return nil
	}
	return event
}

func (s *ScreenAppResourcesList) showSearchBar() {
	s.filteredResults = s.resources
	s.fillTable(s.resources)
	s.searchBar.InputField.SetText("")
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, 1, -1)
	s.grid.AddItem(s.searchBar.InputField, 1, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 2, 0, 1, 1, 0, 0, true)
	s.app.SetFocus(s.searchBar.InputField)
}

func (s *ScreenAppResourcesList) hideSearchBar() {
	s.grid.RemoveItem(s.searchBar.InputField)
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, -1)
	s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)
	s.searchBar.InputField.SetText("")
	s.app.SetFocus(s.table)
}

func (s *ScreenAppResourcesList) fillTable(resources []argocd.Resource) {
	s.table.Clear()
	headers := []string{"Kind", "Name", "Health", "SyncStatus", "Namespace"}
	for col, header := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", header)).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, cell)
	}
	row := 1
	for _, res := range resources {
		kindCell := tview.NewTableCell(res.Kind).SetExpansion(1)
		nameCell := tview.NewTableCell(res.Name).SetExpansion(1)
		healthStatusCell := tview.NewTableCell(res.HealthStatus).SetExpansion(1)
		syncStatusCell := tview.NewTableCell(res.SyncStatus).SetExpansion(1)
		namespaceCell := tview.NewTableCell(res.Namespace).SetExpansion(1)
		s.table.SetCell(row, 0, kindCell)
		s.table.SetCell(row, 1, nameCell)
		s.table.SetCell(row, 2, healthStatusCell)
		s.table.SetCell(row, 3, syncStatusCell)
		s.table.SetCell(row, 4, namespaceCell)

		color := common.ColorForHealthStatus(res.HealthStatus)
		common.SetRowColor(s.table, row, len(headers), color)

		row++
	}
}

func (s *ScreenAppResourcesList) Name() string {
	return "ApplicationResourcesList"
}

var _ ui.Screen = (*ScreenAppResourcesList)(nil)
