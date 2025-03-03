package applicationResourcesList

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
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TreeResource struct {
	Kind       string
	Name       string
	Health     string
	SyncStatus string
	Namespace  string

	Children []*TreeResource
	Expanded bool
	Depth    int
	IsLast   bool
}

type TreeLineInfo struct {
	LineChars []rune
}

type ScreenAppResourcesList struct {
	app          *tview.Application
	instanceInfo *common.InstanceInfo
	client       *argocd.ArgoCdClient
	router       *ui.Router

	table     *tview.Table
	grid      *tview.Grid
	pages     *tview.Pages
	searchBar *components.SimpleSearchBar

	resources        []argocd.Resource
	filteredResults  []argocd.Resource
	rootResources    []*TreeResource
	visibleResources []*TreeResource
	selectedAppName  string
	allExpanded      bool

	originalNodes map[string]*TreeResource

	topBar        *TopBar
	tableView     *TableView
	footer        *Footer
	helpView      *components.HelpView
	filterManager *filters.ResourceFilterManager

	appHealthStatus string
	appSyncStatus   string
}

func New(
	app *tview.Application,
	resources []argocd.Resource,
	selectedAppName string,
	r *ui.Router,
	instanceInfo *common.InstanceInfo,
	client *argocd.ArgoCdClient,
) *ScreenAppResourcesList {
	var healthStatus, syncStatus string

	apps, err := client.GetApps()
	if err == nil {
		for _, app := range apps {
			if app.Name == selectedAppName {
				healthStatus = app.HealthStatus
				syncStatus = app.SyncStatus
				break
			}
		}
	}

	instanceInfo = instanceInfo.WithAppInfo(selectedAppName, healthStatus, syncStatus)

	return &ScreenAppResourcesList{
		app:             app,
		instanceInfo:    instanceInfo,
		client:          client,
		router:          r,
		resources:       resources,
		filteredResults: resources,
		selectedAppName: selectedAppName,
		originalNodes:   make(map[string]*TreeResource),
		allExpanded:     true,
		appHealthStatus: healthStatus,
		appSyncStatus:   syncStatus,
	}
}

func expandFully(node *TreeResource) {
	node.Expanded = true
	for _, child := range node.Children {
		expandFully(child)
	}
}

func collapseFully(node *TreeResource) {
	node.Expanded = false
	for _, child := range node.Children {
		collapseFully(child)
	}
}

func getNodeKey(node *TreeResource) string {
	return fmt.Sprintf("%s|%s|%s", node.Kind, node.Namespace, node.Name)
}

func markLastNodes(resources []*TreeResource) {
	for i, r := range resources {
		r.IsLast = (i == len(resources)-1)
		markLastNodes(r.Children)
	}
}

func flattenResourcesWithLines(resources []*TreeResource, depth int, parentLineInfo *TreeLineInfo) []*TreeResource {
	var result []*TreeResource

	for i, r := range resources {
		isLast := (i == len(resources)-1)
		r.Depth = depth
		r.IsLast = isLast

		result = append(result, r)

		if r.Expanded && len(r.Children) > 0 {
			var childLineInfo *TreeLineInfo
			if parentLineInfo == nil {
				childLineInfo = &TreeLineInfo{LineChars: make([]rune, depth+1)}
			} else {
				childLineInfo = &TreeLineInfo{LineChars: make([]rune, depth+1)}
				copy(childLineInfo.LineChars, parentLineInfo.LineChars)
			}

			if isLast {
				childLineInfo.LineChars[depth] = ' '
			} else {
				childLineInfo.LineChars[depth] = '│'
			}

			childNodes := flattenResourcesWithLines(r.Children, depth+1, childLineInfo)
			result = append(result, childNodes...)
		}
	}

	return result
}

func (s *ScreenAppResourcesList) extractRootKindFilters() map[string]bool {
	rootKindTypes := make(map[string]bool)
	for _, root := range s.rootResources {
		rootKindTypes[root.Kind] = true
	}
	return rootKindTypes
}

func (s *ScreenAppResourcesList) Init() tview.Primitive {
	textColor := tcell.NewHexColor(0x00bebe)        // Цвет текста (#00bebe)
	backgroundColor := tcell.NewHexColor(0x000000)  // Цвет фона (#000000)
	borderColor := tcell.NewHexColor(0x63a0bf)      // Цвет границы (#63a0bf)
	shortcutKeyColor := tcell.NewHexColor(0x017be9) // Цвет клавиш (#017be9)
	selectedBgColor := tcell.NewHexColor(0x373737)  // Цвет выделения (#373737)

	s.topBar = NewTopBar(s.instanceInfo, s.selectedAppName, backgroundColor, shortcutKeyColor, textColor)
	s.footer = NewFooter(s.app, backgroundColor, shortcutKeyColor)
	s.tableView = NewTableView(s.selectedAppName, textColor, borderColor, backgroundColor, selectedBgColor)

	topBarPrimitive := s.topBar.Init()
	footerPrimitive := s.footer.Init()

	s.searchBar = components.NewSimpleSearchBar("🔍 ", 0)
	s.initLiveSearch()
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

	s.table = s.tableView.Init()

	s.grid = tview.NewGrid().
		SetRows(3, 0, 1). // header (topBar), table, footer
		SetColumns(0).
		SetBorders(true)
	s.grid.AddItem(topBarPrimitive, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 0, 1, 1, 0, 0, true).
		AddItem(footerPrimitive, 2, 0, 1, 1, 0, 0, false)

	s.pages = tview.NewPages().
		AddPage("main", s.grid, true, true)

	s.filterManager = filters.NewResourceFilterManager(s.app, s.pages)
	s.filterManager.SetFilterChangedHandler(s.onFiltersChanged)

	s.helpView = components.NewHelpView()
	s.helpView.View.SetInputCapture(s.helpView.GetInputCapture(func() {
		s.pages.SwitchToPage("main")
		s.app.SetFocus(s.table)
	}))
	s.pages.AddPage("help", s.helpView.View, true, false)

	s.table.SetInputCapture(s.onTableKey)

	if err := s.buildTreeFromResourceTree(); err != nil {
		s.showToast(fmt.Sprintf("Error building tree: %v", err), 3*time.Second)
	}

	s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)

	rootKindTypes := s.extractRootKindFilters()

	healthStatuses := make(map[string]bool)
	syncStatuses := make(map[string]bool)
	var collectStatuses func([]*TreeResource)
	collectStatuses = func(nodes []*TreeResource) {
		for _, node := range nodes {
			if node.Health != "" {
				healthStatuses[node.Health] = true
			}
			if node.SyncStatus != "" {
				syncStatuses[node.SyncStatus] = true
			}
			collectStatuses(node.Children)
		}
	}
	collectStatuses(s.rootResources)

	healthStatusList := make([]string, 0, len(healthStatuses))
	for status := range healthStatuses {
		healthStatusList = append(healthStatusList, status)
	}
	sort.Strings(healthStatusList)

	syncStatusList := make([]string, 0, len(syncStatuses))
	for status := range syncStatuses {
		syncStatusList = append(syncStatusList, status)
	}
	sort.Strings(syncStatusList)

	s.filterManager = filters.NewResourceFilterManager(s.app, s.pages)
	s.filterManager.SetFilterChangedHandler(s.onFiltersChanged)
	s.filterManager.ExtractKindsFromResources(rootKindTypes)
	s.filterManager.SetHealthStatuses(healthStatusList)
	s.filterManager.SetSyncStatuses(syncStatusList)

	s.fillTableTreeMode()
	return s.pages
}

func (s *ScreenAppResourcesList) onFiltersChanged(activeFilters []filters.FilterState) {
	var kindFilter, healthFilter, syncFilter string

	for _, filter := range activeFilters {
		switch filter.Type {
		case filters.ResourceKindFilter:
			kindFilter = filter.Value
		case filters.HealthFilter:
			healthFilter = filter.Value
		case filters.SyncFilter:
			syncFilter = filter.Value
		}
	}

	_ = s.buildTreeFromResourceTree()

	if kindFilter != "" {
		s.filterResourcesByKind(kindFilter)
	} else if healthFilter != "" || syncFilter != "" {
		s.filterResourcesByStatus(healthFilter, syncFilter)
	} else {
		s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)
	}

	s.fillTableTreeMode()
}

func (s *ScreenAppResourcesList) filterResourcesByKind(kindFilter string) {
	if kindFilter == "" {
		s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)
		return
	}

	var filteredRoots []*TreeResource

	for _, root := range s.rootResources {
		if root.Kind == kindFilter {
			rootCopy := *root
			rootCopy.Expanded = true
			filteredRoots = append(filteredRoots, &rootCopy)
		}
	}

	if len(filteredRoots) > 0 {
		s.visibleResources = flattenResourcesWithLines(filteredRoots, 0, nil)
	} else {
		s.showToast(fmt.Sprintf("No root resources of type %s found", kindFilter), 2*time.Second)
		s.filterManager.SetFilter(filters.ResourceKindFilter, "")
		s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)
	}
}

func (s *ScreenAppResourcesList) filterResourcesByStatus(healthFilter, syncFilter string) {
	if healthFilter == "" && syncFilter == "" {
		s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)
		return
	}

	allNodes := flattenResourcesWithLines(s.rootResources, 0, nil)
	var filtered []*TreeResource

	for _, node := range allNodes {
		healthMatch := healthFilter == "" || strings.EqualFold(node.Health, healthFilter)
		syncMatch := syncFilter == "" || strings.EqualFold(node.SyncStatus, syncFilter)

		if healthMatch && syncMatch {
			filtered = append(filtered, node)
		}
	}

	s.visibleResources = filtered
}

func (s *ScreenAppResourcesList) toggleExpansionAll() {
	if s.allExpanded {
		for _, root := range s.rootResources {
			collapseFully(root)
		}
		s.allExpanded = false
	} else {
		for _, root := range s.rootResources {
			expandFully(root)
		}
		s.allExpanded = true
	}

	s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)
	s.fillTableTreeMode()
}

func (s *ScreenAppResourcesList) initLiveSearch() {
	var debounceTimer *time.Timer
	s.searchBar.InputField.SetChangedFunc(func(text string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
			s.app.QueueUpdateDraw(func() {
				s.filterResources(text)
			})
		})
	})
}

func (s *ScreenAppResourcesList) showSearchBar() {
	s.grid.RemoveItem(s.table)
	s.searchBar.InputField.SetText("")

	s.grid.SetRows(3, 1, -1, 1)
	s.grid.AddItem(s.searchBar.InputField, 1, 0, 1, 1, 0, 0, false)
	s.grid.AddItem(s.table, 2, 0, 1, 1, 0, 0, true)
	s.app.SetFocus(s.searchBar.InputField)
}

func (s *ScreenAppResourcesList) hideSearchBar() {
	s.grid.RemoveItem(s.searchBar.InputField)
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, 0, 1)
	s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)
	s.app.SetFocus(s.table)
}

// Восстановленный метод для обработки завершения поиска
func (s *ScreenAppResourcesList) searchDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		query := s.searchBar.InputField.GetText()
		s.filterResources(query)
		s.hideSearchBar()
		s.app.SetFocus(s.table)
	}
}

func (s *ScreenAppResourcesList) filterResources(query string) {
	if query == "" {
		_ = s.buildTreeFromResourceTree()
		s.fillTableTreeMode()
		return
	}

	lq := strings.ToLower(query)

	_ = s.buildTreeFromResourceTree()
	allNodes := flattenResourcesWithLines(s.rootResources, 0, nil)
	var filtered []*TreeResource
	for _, n := range allNodes {
		text := strings.ToLower(n.Kind + " " + n.Name + " " + n.Namespace + " " + n.Health + " " + n.SyncStatus)
		if strings.Contains(text, lq) {
			filtered = append(filtered, n)
		}
	}

	s.footer.UpdateResourceCount(len(filtered))

	s.tableView.FillTableWithSearch(filtered)
}

func (s *ScreenAppResourcesList) buildTreeFromResourceTree() error {
	appTree, err := s.client.GetResourceTree(s.selectedAppName)
	if err != nil {
		return err
	}
	s.rootResources = buildTreeFromNodes(appTree.Nodes)
	markLastNodes(s.rootResources)
	s.buildOriginalNodesMap()
	return nil
}

func buildTreeFromNodes(nodes []v1alpha1.ResourceNode) []*TreeResource {
	resourceMap := make(map[string]*TreeResource)
	for i := range nodes {
		n := &nodes[i]
		tr := &TreeResource{
			Kind:      n.Kind,
			Name:      n.Name,
			Namespace: n.Namespace,
			Expanded:  true,
			Children:  []*TreeResource{},
		}
		if n.Health != nil {
			tr.Health = string(n.Health.Status)
		} else {
			tr.Health = "Unknown"
		}
		tr.SyncStatus = "Synced"
		resourceMap[n.UID] = tr
	}

	for i := range nodes {
		n := &nodes[i]
		childTR := resourceMap[n.UID]
		for _, pref := range n.ParentRefs {
			if parentTR, ok := resourceMap[pref.UID]; ok {
				parentTR.Children = append(parentTR.Children, childTR)
			}
		}
	}

	var roots []*TreeResource
	for i := range nodes {
		n := &nodes[i]
		if len(n.ParentRefs) == 0 {
			roots = append(roots, resourceMap[n.UID])
		}
	}
	return roots
}

func (s *ScreenAppResourcesList) onTableKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'q':
		s.app.Stop()
		return nil
	case 'b':
		s.router.Back()
		return nil
	case 't':
		s.toggleExpansionAll()
		return nil
	case '/', ':':
		s.showSearchBar()
		return nil
	case '?':
		s.pages.SwitchToPage("help")
		s.app.SetFocus(s.helpView.View)
		return nil
	case 'f', 'F':
		s.filterManager.ShowFilterMenu()
		return nil
	case 'd', 'D':
		s.filterManager.ToggleFilter(filters.ResourceKindFilter, "Deployment")
		return nil
	case 's', 'S':
		s.filterManager.ToggleFilter(filters.ResourceKindFilter, "Service")
		return nil
	case 'i', 'I':
		s.filterManager.ToggleFilter(filters.ResourceKindFilter, "Ingress")
		return nil
	case 'c', 'C':
		if event.Rune() == 'C' {
			s.filterManager.ClearFilters()
		} else {
			s.filterManager.ToggleFilter(filters.ResourceKindFilter, "ConfigMap")
		}
		return nil
	case 'h', 'H':
		s.filterManager.ToggleFilter(filters.HealthFilter, "Healthy")
		return nil
	case 'p', 'P':
		s.filterManager.ToggleFilter(filters.HealthFilter, "Progressing")
		return nil
	case 'o', 'O':
		s.filterManager.ToggleFilter(filters.SyncFilter, "OutOfSync")
		return nil
	}

	if event.Key() == tcell.KeyEnter {
		row, _ := s.table.GetSelection()
		if row > 0 && row-1 < len(s.visibleResources) {
			selected := s.visibleResources[row-1]
			nodeKey := getNodeKey(selected)

			if originalNode, exists := s.originalNodes[nodeKey]; exists {
				if len(originalNode.Children) > 0 {
					if !originalNode.Expanded {
						expandFully(originalNode)
					} else {
						collapseFully(originalNode)
					}

					s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)
					s.fillTableTreeMode()
				}
			}
		}
		return nil
	}

	return event
}

func (s *ScreenAppResourcesList) buildOriginalNodesMap() {
	s.originalNodes = make(map[string]*TreeResource)

	var addToMap func([]*TreeResource)
	addToMap = func(nodes []*TreeResource) {
		for _, node := range nodes {
			s.originalNodes[getNodeKey(node)] = node
			addToMap(node.Children)
		}
	}

	addToMap(s.rootResources)
}

func (s *ScreenAppResourcesList) fillTableTreeMode() {
	s.footer.UpdateResourceCount(len(s.visibleResources))
	activeFiltersText := s.filterManager.GetActiveFiltersText()
	s.tableView.FillTableWithTree(s.visibleResources, activeFiltersText)
}

func (s *ScreenAppResourcesList) showToast(message string, duration time.Duration) {
	var toast *components.SimpleSearchBar
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, 1, -1, 1) // topBar, toast, table, footer
	toast = components.NewSimpleSearchBar("✅  ", 0)
	toast.InputField.SetText(message)
	s.grid.AddItem(toast.InputField, 1, 0, 1, 1, 0, 0, false)
	s.grid.AddItem(s.table, 2, 0, 1, 1, 0, 0, true)

	go func() {
		time.Sleep(duration)
		s.app.QueueUpdateDraw(func() {
			s.grid.RemoveItem(toast.InputField)
			s.grid.RemoveItem(s.table)
			s.grid.SetRows(3, 0, 1) // topBar, table, footer
			s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)
		})
	}()
}

func (s *ScreenAppResourcesList) Name() string {
	return "ApplicationResourcesList"
}
