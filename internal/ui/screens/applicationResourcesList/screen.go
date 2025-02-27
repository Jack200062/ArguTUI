package applicationResourcesList

import (
	"fmt"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ScreenAppResourcesList struct {
	app          *tview.Application
	appName      string
	resources    []argocd.Resource
	router       *ui.Router
	instanceInfo *common.InstanceInfo

	table             *tview.Table
	grid              *tview.Grid
	pages             *tview.Pages
	searchBar         *components.SearchBar
	filteredResources []argocd.Resource
}

func New(app *tview.Application, resources []argocd.Resource, appName string, r *ui.Router, instanceInfo *common.InstanceInfo) *ScreenAppResourcesList {
	return &ScreenAppResourcesList{
		app:               app,
		appName:           appName,
		resources:         resources,
		router:            r,
		filteredResources: resources,
		instanceInfo:      instanceInfo,
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

	resourcesBoxTitle := fmt.Sprintf(" Resources %s app ", s.appName)
	s.table = tview.NewTable().
		SetSelectable(true, false).
		SetBorders(false)
	s.table.Box.SetBorder(true)
	s.table.Box.SetTitle(resourcesBoxTitle)

	s.searchBar = components.NewSearchBar("=> ", 0)
	s.searchBar.Filter = func(query string) []interface{} {
		var items []interface{}
		for _, resource := range s.resources {
			items = append(items, resource)
		}
		indices := components.FuzzyFilter(query, items)
		filtered := make([]interface{}, len(indices))
		for i, idx := range indices {
			filtered[i] = s.resources[idx]
		}
		return filtered
	}
	s.searchBar.OnDone = func(filtered []interface{}) {
		if filtered == nil {
			s.filteredResources = s.resources
		} else {
			s.filteredResources = make([]argocd.Resource, len(filtered))
			for i, item := range filtered {
				s.filteredResources[i] = item.(argocd.Resource)
			}
		}
		s.fillTable(s.filteredResources)
		s.hideSearchBar()
		s.app.SetFocus(s.table)
	}

	s.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(0).
		SetBorders(true)

	s.grid.AddItem(topBar, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 0, 1, 1, 0, 0, true)

	s.fillTable(s.filteredResources)

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

func (s *ScreenAppResourcesList) onGridKey(event *tcell.EventKey) *tcell.EventKey {
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
	case 'b':
		s.router.Back()
		return nil
	}
	return event
}

func (s *ScreenAppResourcesList) showSearchBar() {
	s.filteredResources = s.resources
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
	headers := []string{"Kind", "Name", "Namespace"}
	for col, header := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", header)).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, cell)
	}

	row := 1
	for _, res := range resources {
		s.table.SetCell(row, 0, tview.NewTableCell(res.Kind).SetExpansion(1))
		s.table.SetCell(row, 1, tview.NewTableCell(res.Name).SetExpansion(1))
		s.table.SetCell(row, 2, tview.NewTableCell(res.Namespace).SetExpansion(1))
		row++
	}
}

func (s *ScreenAppResourcesList) Name() string {
	return "ApplicationResourcesList"
}

var _ ui.Screen = (*ScreenAppResourcesList)(nil)
