package applicationlist

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/components"
	"github.com/Jack200062/ArguTUI/internal/ui/filters"
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
	searchBar    *components.SimpleSearchBar
	filteredApps []argocd.Application

	projectFilter   string
	healthFilter    string
	syncFilter      string
	searchQuery     string
	lastRefreshTime time.Time

	topBar    *TopBar
	footer    *Footer
	tableView *TableView
}

func New(
	app *tview.Application,
	c *argocd.ArgoCdClient,
	r *ui.Router,
	instanceInfo *common.InstanceInfo,
	apps []argocd.Application,
) *ScreenAppList {
	if instanceInfo == nil {
		instanceInfo = common.NewInstanceInfo("unknown", "unknown")
	}

	return &ScreenAppList{
		app:             app,
		client:          c,
		router:          r,
		instanceInfo:    instanceInfo,
		apps:            apps,
		filteredApps:    apps,
		lastRefreshTime: time.Now(),
	}
}

func (s *ScreenAppList) getApplicationStats() (healthy, degraded, outOfSync int) {
	for _, app := range s.apps {
		if strings.EqualFold(app.HealthStatus, "Healthy") {
			healthy++
		}
		if strings.EqualFold(app.HealthStatus, "Degraded") {
			degraded++
		}
		if strings.EqualFold(app.SyncStatus, "OutOfSync") {
			outOfSync++
		}
	}
	return
}

func (s *ScreenAppList) Init() tview.Primitive {
	s.startAutoRefresh()

	textColor := tcell.NewHexColor(0x00bebe)        // table Title Color
	backgroundColor := tcell.NewHexColor(0x000000)  // background Color
	borderColor := tcell.NewHexColor(0x63a0bf)      // table border Color
	shortcutKeyColor := tcell.NewHexColor(0x017be9) // shortCutKey Color
	selectedBgColor := tcell.NewHexColor(0x373737)  // row background Color

	s.topBar = NewTopBar(s.instanceInfo, backgroundColor, shortcutKeyColor, textColor)
	s.footer = NewFooter(s.app, backgroundColor, shortcutKeyColor)
	s.tableView = NewTableView(textColor, borderColor, backgroundColor, selectedBgColor)

	topBarPrimitive := s.topBar.Init()

	healthy, degraded, outOfSync := s.getApplicationStats()
	s.topBar.UpdateStats(healthy, degraded, outOfSync)
	footerPrimitive := s.footer.Init()
	s.footer.UpdateTimeInfo(s.lastRefreshTime)

	s.searchBar = components.NewSimpleSearchBar("🐙 ", 0)
	s.initLiveSearch()
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

	s.table = s.tableView.Init()
	s.tableView.FillTable(s.filteredApps, s.getActiveFiltersText())

	s.grid = tview.NewGrid().
		SetRows(4, 0, 1). // header (topbar), table, footer
		SetColumns(0).
		SetBorders(true)
	s.grid.AddItem(topBarPrimitive, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 0, 1, 1, 0, 0, true).
		AddItem(footerPrimitive, 2, 0, 1, 1, 0, 0, false)

	s.pages = tview.NewPages().
		AddPage("main", s.grid, true, true)

	helpView := components.NewHelpView()
	s.pages.AddPage("help", helpView.View, true, false)

	s.grid.SetInputCapture(s.onGridKey)

	return s.pages
}

func (s *ScreenAppList) startAutoRefresh() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			s.app.QueueUpdateDraw(func() {
				s.refreshApps()
			})
		}
	}()
}

func (s *ScreenAppList) refreshApps() {
	newApps, err := s.client.GetApps()
	if err != nil {
		return
	}
	s.lastRefreshTime = time.Now()
	s.apps = newApps
	s.applyFilters()

	healthy, degraded, outOfSync := s.getApplicationStats()
	s.topBar.UpdateStats(healthy, degraded, outOfSync)
	s.footer.UpdateTimeInfo(s.lastRefreshTime)
}

func (s *ScreenAppList) applyFilters() {
	filteredApps := s.apps

	if s.projectFilter != "" {
		var filtered []argocd.Application
		for _, app := range filteredApps {
			if app.Project == s.projectFilter {
				filtered = append(filtered, app)
			}
		}
		filteredApps = filtered
	}

	if s.healthFilter != "" {
		var filtered []argocd.Application
		for _, app := range filteredApps {
			if strings.EqualFold(app.HealthStatus, s.healthFilter) {
				filtered = append(filtered, app)
			}
		}
		filteredApps = filtered
	}

	if s.syncFilter != "" {
		var filtered []argocd.Application
		for _, app := range filteredApps {
			if strings.EqualFold(app.SyncStatus, s.syncFilter) {
				filtered = append(filtered, app)
			}
		}
		filteredApps = filtered
	}

	if s.searchQuery != "" {
		filteredApps = filterApplications(filteredApps, s.searchQuery)
	}

	s.filteredApps = filteredApps
	s.tableView.FillTable(s.filteredApps, s.getActiveFiltersText())
}

func (s *ScreenAppList) getActiveFiltersText() string {
	var parts []string

	if s.projectFilter != "" {
		parts = append(parts, fmt.Sprintf("Project=%s", s.projectFilter))
	}

	if s.healthFilter != "" {
		parts = append(parts, fmt.Sprintf("Health=%s", s.healthFilter))
	}

	if s.syncFilter != "" {
		parts = append(parts, fmt.Sprintf("Sync=%s", s.syncFilter))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}

func (s *ScreenAppList) initLiveSearch() {
	var debounceTimer *time.Timer
	s.searchBar.InputField.SetChangedFunc(func(text string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
			s.app.QueueUpdateDraw(func() {
				s.searchQuery = text
				s.applyFilters()
			})
		})
	})
}

func (s *ScreenAppList) showSearchBar() {
	s.searchBar.InputField.SetText("")
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(4, 1, -1, 1) // topBar, searchBar, table, footer
	s.grid.AddItem(s.searchBar.InputField, 1, 0, 1, 1, 0, 0, false)
	s.grid.AddItem(s.table, 2, 0, 1, 1, 0, 0, true)
	s.app.SetFocus(s.searchBar.InputField)
}

func (s *ScreenAppList) hideSearchBar() {
	s.grid.RemoveItem(s.searchBar.InputField)
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(4, -1, 1) // topBar, table, footer
	s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)
	s.app.SetFocus(s.table)
}

func (s *ScreenAppList) searchDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		s.searchQuery = s.searchBar.InputField.GetText()
		s.applyFilters()
		s.hideSearchBar()
		s.app.SetFocus(s.table)
	}
}

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

func (s *ScreenAppList) onGridKey(event *tcell.EventKey) *tcell.EventKey {
	if s.app.GetFocus() == s.searchBar.InputField {
		return event
	}
	switch event.Rune() {
	case 'I':
		s.router.SwitchTo("InstanceSelection")
		return nil
	case '?':
		s.pages.ShowPage("help")
		return nil
	case 'q':
		if s.footer != nil {
			s.footer.Stop()
		}
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
	case 'S':
		row, _ := s.table.GetSelection()
		if row < 1 || row-1 >= len(s.filteredApps) {
			return event
		}
		selectedApp := s.filteredApps[row-1]
		err := s.syncApplication(selectedApp.Name)
		if err != nil {
			modal := components.ErrorModal(
				fmt.Sprintf("Error syncing app %s:", selectedApp.Name),
				err.Error(),
				s.modalClose,
			)
			s.app.SetRoot(modal, true)
		}
		return nil
	case 'D':
		row, _ := s.table.GetSelection()
		if row < 1 || row-1 >= len(s.filteredApps) {
			return event
		}
		selectedApp := s.filteredApps[row-1]
		s.confirmAndDeleteApplication(selectedApp.Name)
		return nil
	case 'F', 'f':
		s.showFilterMenu()
		return nil
	case 'h', 'H':
		if s.healthFilter == "Healthy" {
			s.healthFilter = ""
		} else {
			s.healthFilter = "Healthy"
		}
		s.applyFilters()
		return nil
	case 'p', 'P':
		if s.healthFilter == "Progressing" {
			s.healthFilter = ""
		} else {
			s.healthFilter = "Progressing"
		}
		s.applyFilters()
		return nil
	case 's':
		if s.syncFilter == "Synced" {
			s.syncFilter = ""
		} else {
			s.syncFilter = "Synced"
		}
		s.applyFilters()
		return nil
	case 'o', 'O':
		if s.syncFilter == "OutOfSync" {
			s.syncFilter = ""
		} else {
			s.syncFilter = "OutOfSync"
		}
		s.applyFilters()
		return nil
	case 'c', 'C':
		s.projectFilter = ""
		s.healthFilter = ""
		s.syncFilter = ""
		s.searchQuery = ""
		s.applyFilters()
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
		resScreen := applicationResourcesList.New(s.app, resources, selectedApp.Name, s.router, s.instanceInfo, s.client)
		s.router.AddScreen(resScreen)
		s.router.SwitchTo(resScreen.Name())
		return nil
	}
	return event
}

func (s *ScreenAppList) syncApplication(appName string) error {
	err := s.client.SyncApp(appName)
	if err != nil {
		modal := components.ErrorModal(
			fmt.Sprintf("Error syncing app %s:", appName),
			err.Error(),
			s.modalClose,
		)
		s.app.SetRoot(modal, true)
		return err
	}
	s.showToast(fmt.Sprintf("App %s synced successfully!", appName), 2*time.Second)
	s.refreshApps()
	return nil
}

func (s *ScreenAppList) confirmAndDeleteApplication(appName string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure you want to delete application %s?", appName)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 { // "Yes" button
				err := s.client.DeleteApp(appName)
				if err != nil {
					errorModal := components.ErrorModal(
						fmt.Sprintf("Error deleting app %s:", appName),
						err.Error(),
						s.modalClose,
					)
					s.app.SetRoot(errorModal, true)
					return
				}
				s.showToast(fmt.Sprintf("App %s deleted successfully!", appName), 2*time.Second)
				s.refreshApps()
			}
			s.app.SetRoot(s.pages, true)
		})

	modal.SetBackgroundColor(tcell.ColorDarkRed)
	s.app.SetRoot(modal, true)
}

func (s *ScreenAppList) modalClose() {
	s.app.SetRoot(s.pages, true)
}

func (s *ScreenAppList) showToast(message string, duration time.Duration) {
	var toast *components.SimpleSearchBar
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(4, 1, -1, 1) // topBar, toast, table, footer
	toast = components.NewSimpleSearchBar("✅  ", 0)
	toast.InputField.SetText(message)
	s.grid.AddItem(toast.InputField, 1, 0, 1, 1, 0, 0, false)
	s.grid.AddItem(s.table, 2, 0, 1, 1, 0, 0, true)
	go func() {
		time.Sleep(duration)
		s.app.QueueUpdateDraw(func() {
			s.grid.RemoveItem(toast.InputField)
			s.grid.RemoveItem(s.table)
			s.grid.SetRows(4, 0, 1) // topBar, table, footer
			s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)
		})
	}()
}

func (s *ScreenAppList) showFilterMenu() {
	projects := make(map[string]bool)
	healthStatuses := make(map[string]bool)
	syncStatuses := make(map[string]bool)

	for _, app := range s.apps {
		projects[app.Project] = true
		healthStatuses[app.HealthStatus] = true
		syncStatuses[app.SyncStatus] = true
	}

	projectList := make([]string, 0, len(projects))
	for project := range projects {
		projectList = append(projectList, project)
	}
	sort.Strings(projectList)

	healthList := make([]string, 0, len(healthStatuses))
	for status := range healthStatuses {
		healthList = append(healthList, status)
	}
	sort.Strings(healthList)

	syncList := make([]string, 0, len(syncStatuses))
	for status := range syncStatuses {
		syncList = append(syncList, status)
	}
	sort.Strings(syncList)

	filterCategories := []filters.FilterCategory{
		{
			Title:     "Project",
			Type:      filters.ProjectFilter,
			Options:   projectList,
			Shortcuts: map[string]rune{},
		},
		{
			Title:     "Health Status",
			Type:      filters.HealthFilter,
			Options:   healthList,
			Shortcuts: filters.StandardHealthShortcuts(),
		},
		{
			Title:     "Sync Status",
			Type:      filters.SyncFilter,
			Options:   syncList,
			Shortcuts: filters.StandardSyncShortcuts(),
		},
	}

	activeFilters := []filters.FilterState{}

	if s.projectFilter != "" {
		activeFilters = append(activeFilters, filters.FilterState{
			Type:  filters.ProjectFilter,
			Value: s.projectFilter,
		})
	}

	if s.healthFilter != "" {
		activeFilters = append(activeFilters, filters.FilterState{
			Type:  filters.HealthFilter,
			Value: s.healthFilter,
		})
	}

	if s.syncFilter != "" {
		activeFilters = append(activeFilters, filters.FilterState{
			Type:  filters.SyncFilter,
			Value: s.syncFilter,
		})
	}

	filters.ShowMultiFilterMenu(
		s.app,
		filters.MultiFilterOptions{
			Title:         "FILTER APPLICATIONS",
			ActiveFilters: activeFilters,
			Categories:    filterCategories,
			AsOverlay:     true,
			Position: filters.OverlayPosition{
				Top:    3,  // Padding from top
				Left:   10, // Padding from left
				Right:  10, // Padding from right
				Bottom: 5,  // Padding from bottom
			},
		},
		s.pages,
		func(result filters.FilterResult) {
			if !result.Canceled {
				s.projectFilter = ""
				s.healthFilter = ""
				s.syncFilter = ""

				for _, filter := range result.Filters {
					switch filter.Type {
					case filters.ProjectFilter:
						s.projectFilter = filter.Value
					case filters.HealthFilter:
						s.healthFilter = filter.Value
					case filters.SyncFilter:
						s.syncFilter = filter.Value
					}
				}

				s.applyFilters()
			}
		},
	)
}

func (s *ScreenAppList) Name() string {
	return "ApplicationList"
}

var _ ui.Screen = (*ScreenAppList)(nil)
