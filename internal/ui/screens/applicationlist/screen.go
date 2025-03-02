package applicationlist

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

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
	searchBar    *components.SimpleSearchBar
	filteredApps []argocd.Application

	projectFilter string
	healthFilter  string
	syncFilter    string
	searchQuery   string
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

	shortCutInfo := tview.NewTextView().
		SetText(" q Quit ? Help f Filter h/d/p Health s/o Sync \n R Refresh r RefreshApp c ClearFilters").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorYellow)

	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String()).
		SetTextAlign(tview.AlignLeft)
	instanceBox.SetBorder(false)

	topBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceBox, 0, 1, false).
		AddItem(shortCutInfo, 0, 1, false)

	s.searchBar = components.NewSimpleSearchBar("=> ", 0)
	s.initLiveSearch()
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

	s.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	s.table.SetBorder(true).SetTitle(" Applications ")

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
				s.applyFilters()
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
	s.applyFilters()
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

	// Применяем фильтр по sync status
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
	s.fillTable(s.filteredApps)
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
		return "None"
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

func (s *ScreenAppList) helpInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Rune() == 'q' {
		s.pages.HidePage("help")
		s.app.SetFocus(s.grid)
		return nil
	}
	return event
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

func (s *ScreenAppList) fillTable(apps []argocd.Application) {
	s.table.Clear()
	headers := []string{"Name", "HealthStatus", "SyncStatus", "SyncCommit", "Project", "LastActivity"}
	for col, h := range headers {
		headerCell := tview.NewTableCell(fmt.Sprintf("[::b]%s", h)).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, headerCell)
	}

	title := " Applications "
	if s.projectFilter != "" || s.healthFilter != "" || s.syncFilter != "" || s.searchQuery != "" {
		title = fmt.Sprintf(" Applications (%s) ", s.getActiveFiltersText())
	}
	s.table.SetTitle(title)

	row := 1
	for _, app := range apps {
		nameCell := tview.NewTableCell(app.Name).SetExpansion(1)
		healthStatusCell := tview.NewTableCell(app.HealthStatus).SetExpansion(1)
		syncStatusCell := tview.NewTableCell(app.SyncStatus).SetExpansion(1)
		syncCommitCell := tview.NewTableCell(app.SyncCommit).SetExpansion(1)
		projectCell := tview.NewTableCell(app.Project).SetExpansion(1)
		lastActivityCell := tview.NewTableCell(app.LastActivity).SetExpansion(1)

		s.table.SetCell(row, 0, nameCell)
		s.table.SetCell(row, 1, healthStatusCell)
		s.table.SetCell(row, 2, syncStatusCell)
		s.table.SetCell(row, 3, syncCommitCell)
		s.table.SetCell(row, 4, projectCell)
		s.table.SetCell(row, 5, lastActivityCell)

		rowColor := common.RowColorForStatuses(app.HealthStatus, app.SyncStatus)
		common.SetRowColor(s.table, row, len(headers), rowColor)

		row++
	}
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

	var shortcutsInfo strings.Builder
	shortcutsInfo.WriteString("Keyboard shortcuts: [a] All ")

	standardHealthShortcuts := map[string]rune{
		"Healthy":     'h',
		"Progressing": 'p',
	}

	standardSyncShortcuts := map[string]rune{
		"Synced":    's',
		"OutOfSync": 'o',
	}

	shortcutsInfo.WriteString("\nHealth: ")
	for _, status := range healthList {
		shortcut, hasShortcut := standardHealthShortcuts[status]
		if hasShortcut {
			shortcutsInfo.WriteString(fmt.Sprintf("[%c] %s ", shortcut, status))
		}
	}

	shortcutsInfo.WriteString("\nSync: ")
	for _, status := range syncList {
		shortcut, hasShortcut := standardSyncShortcuts[status]
		if hasShortcut {
			shortcutsInfo.WriteString(fmt.Sprintf("[%c] %s ", shortcut, status))
		}
	}

	modal := tview.NewModal()
	modalText := fmt.Sprintf("Filter Applications\n\nCurrent filters: %s\n\n%s", s.getActiveFiltersText(), shortcutsInfo.String())
	modal.SetText(modalText)

	buttons := []string{"Clear All Filters", "Project Filter", "Health Filter", "Sync Filter"}
	modal.SetBackgroundColor(tcell.ColorDarkBlue)
	modal.AddButtons(buttons)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		switch buttonIndex {
		case 0: // Clear All Filters
			s.projectFilter = ""
			s.healthFilter = ""
			s.syncFilter = ""
		case 1: // Project Filter
			s.showProjectFilterMenu(projectList)
			return
		case 2: // Health Filter
			s.showHealthFilterMenu(healthList)
			return
		case 3: // Sync Filter
			s.showSyncFilterMenu(syncList)
			return
		}

		s.applyFilters()
		s.app.SetRoot(s.pages, true)
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			s.app.SetRoot(s.pages, true)
			return nil
		}

		switch event.Rune() {
		case 'a', 'A':
			s.projectFilter = ""
			s.healthFilter = ""
			s.syncFilter = ""
			s.applyFilters()
			s.app.SetRoot(s.pages, true)
			return nil
		default:
			for status, shortcut := range standardHealthShortcuts {
				if event.Rune() == shortcut || event.Rune() == unicode.ToUpper(shortcut) {
					s.healthFilter = status
					s.applyFilters()
					s.app.SetRoot(s.pages, true)
					return nil
				}
			}

			for status, shortcut := range standardSyncShortcuts {
				if event.Rune() == shortcut || event.Rune() == unicode.ToUpper(shortcut) {
					s.syncFilter = status
					s.applyFilters()
					s.app.SetRoot(s.pages, true)
					return nil
				}
			}
		}

		return event
	})

	s.app.SetRoot(modal, true)
}

func (s *ScreenAppList) showProjectFilterMenu(projects []string) {
	if len(projects) == 0 {
		s.showToast("No projects found", 2*time.Second)
		return
	}

	modal := tview.NewModal()
	modalText := "Select project to filter by:"
	modal.SetText(modalText)

	buttons := append([]string{"All (clear filter)"}, projects...)
	modal.SetBackgroundColor(tcell.ColorDarkBlue)
	modal.AddButtons(buttons)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 {
			s.projectFilter = ""
		} else if buttonIndex > 0 && buttonIndex <= len(projects) {
			s.projectFilter = projects[buttonIndex-1]
		}

		s.applyFilters()
		s.app.SetRoot(s.pages, true)
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			s.app.SetRoot(s.pages, true)
			return nil
		}
		return event
	})

	s.app.SetRoot(modal, true)
}

func (s *ScreenAppList) showHealthFilterMenu(statuses []string) {
	if len(statuses) == 0 {
		s.showToast("No health statuses found", 2*time.Second)
		return
	}

	modal := tview.NewModal()
	modalText := "Select health status to filter by:"
	modal.SetText(modalText)

	buttons := append([]string{"All (clear filter)"}, statuses...)
	modal.SetBackgroundColor(tcell.ColorDarkBlue)
	modal.AddButtons(buttons)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 {
			s.healthFilter = ""
		} else if buttonIndex > 0 && buttonIndex <= len(statuses) {
			s.healthFilter = statuses[buttonIndex-1]
		}

		s.applyFilters()
		s.app.SetRoot(s.pages, true)
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			s.app.SetRoot(s.pages, true)
			return nil
		}
		return event
	})

	s.app.SetRoot(modal, true)
}

func (s *ScreenAppList) showSyncFilterMenu(statuses []string) {
	if len(statuses) == 0 {
		s.showToast("No sync statuses found", 2*time.Second)
		return
	}

	modal := tview.NewModal()
	modalText := "Select sync status to filter by:"
	modal.SetText(modalText)

	buttons := append([]string{"All (clear filter)"}, statuses...)
	modal.SetBackgroundColor(tcell.ColorDarkBlue)
	modal.AddButtons(buttons)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 {
			s.syncFilter = ""
		} else if buttonIndex > 0 && buttonIndex <= len(statuses) {
			s.syncFilter = statuses[buttonIndex-1]
		}

		s.applyFilters()
		s.app.SetRoot(s.pages, true)
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			s.app.SetRoot(s.pages, true)
			return nil
		}
		return event
	})

	s.app.SetRoot(modal, true)
}

func (s *ScreenAppList) Name() string {
	return "ApplicationList"
}

var _ ui.Screen = (*ScreenAppList)(nil)
