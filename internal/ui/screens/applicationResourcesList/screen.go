package applicationResourcesList

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Jack200062/ArguTUI/internal/models"
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

	resources         []models.Resource
	filteredResources []*TreeResource
	rootResources     []*TreeResource
	visibleResources  []*TreeResource
	selectedApp       *models.Application
	allExpanded       bool

	originalNodes map[string]*TreeResource

	topBar    *TopBar
	tableView *TableView
	footer    *Footer
	helpView  *components.HelpView

	appHealthStatus string
	appSyncStatus   string

	filterCategories struct {
		kindList   []string
		healthList []string
		syncList   []string
	}
	activeFilters []filters.Filter
	kindFilter    string
	healthFilter  string
	syncFilter    string
}

func New(
	app *tview.Application,
	resources []models.Resource,
	r *ui.Router,
	instanceInfo *common.InstanceInfo,
	client *argocd.ArgoCdClient,
	selectedApp *models.Application,
) *ScreenAppResourcesList {

	instanceInfo = instanceInfo.WithAppInfo(selectedApp.Name, selectedApp.HealthStatus, selectedApp.SyncStatus)

	resourcesListScreen := &ScreenAppResourcesList{
		app:             app,
		instanceInfo:    instanceInfo,
		client:          client,
		router:          r,
		resources:       resources,
		selectedApp:     selectedApp,
		originalNodes:   make(map[string]*TreeResource),
		allExpanded:     false,
		appHealthStatus: selectedApp.HealthStatus,
		appSyncStatus:   selectedApp.SyncStatus,
	}
	resourcesListScreen.updateFilterCategories()

	return resourcesListScreen
}

func expandFully(node *TreeResource) {
	node.Expanded = true
	for _, child := range node.Children {
		expandFully(child)
	}
}

func expand(node *TreeResource, depth int) {
	node.Expanded = true
	if depth > 0 {
		for _, child := range node.Children {
			expand(child, depth-1)
		}
	}
}

func collapseFully(node *TreeResource) {
	node.Expanded = false
	for _, child := range node.Children {
		collapseFully(child)
	}
}

func collapse(node *TreeResource, depth int) {
	node.Expanded = false
	if depth > 0 {
		for _, child := range node.Children {
			collapse(child, depth-1)
		}
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

func (s *ScreenAppResourcesList) Init() tview.Primitive {
	textColor := tcell.NewHexColor(0x00bebe)        // Цвет текста (#00bebe)
	backgroundColor := tcell.NewHexColor(0x000000)  // Цвет фона (#000000)
	borderColor := tcell.NewHexColor(0x63a0bf)      // Цвет границы (#63a0bf)
	shortcutKeyColor := tcell.NewHexColor(0x017be9) // Цвет клавиш (#017be9)
	selectedBgColor := tcell.NewHexColor(0x373737)  // Цвет выделения (#373737)

	s.topBar = NewTopBar(s.instanceInfo, s.selectedApp.Name, backgroundColor, shortcutKeyColor, textColor)
	s.footer = NewFooter(s.app, backgroundColor, shortcutKeyColor)
	s.tableView = NewTableView(s.selectedApp.Name, textColor, borderColor, backgroundColor, selectedBgColor)

	topBarPrimitive := s.topBar.Init()
	footerPrimitive := s.footer.Init()

	s.searchBar = components.NewSimpleSearchBar("🐙 => ", 0)
	s.initLiveSearch()
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

	s.table = s.tableView.Init()

	s.grid = tview.NewGrid().
		SetRows(4, 0, 1). // header (topBar), table, footer
		SetColumns(0).
		SetBorders(true)
	s.grid.AddItem(topBarPrimitive, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 0, 1, 1, 0, 0, true).
		AddItem(footerPrimitive, 2, 0, 1, 1, 0, 0, false)

	s.pages = tview.NewPages().
		AddPage("main", s.grid, true, true)

	s.helpView = components.NewHelpView()
	s.helpView.Grid.SetInputCapture(s.helpView.GetInputCapture(func() {
		s.pages.SwitchToPage("main")
		s.app.SetFocus(s.table)
	}))
	s.pages.AddPage("help", s.helpView.Grid, true, false)

	s.table.SetInputCapture(s.onTableKey)

	if err := s.buildTreeFromResourceTree(); err != nil {
		s.showToast(fmt.Sprintf("Error building tree: %v", err), 2*time.Second)
	}

	s.visibleResources = flattenResourcesWithLines(s.rootResources, 0, nil)

	s.fillTableTreeMode()
	return s.pages
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
	appTree, resourceStatuses, err := s.client.GetResourceTree(s.selectedApp.Name)
	if err != nil {
		return err
	}
	s.rootResources = buildTreeFromNodes(appTree.Nodes, resourceStatuses)
	markLastNodes(s.rootResources)
	s.buildOriginalNodesMap()
	return nil
}

func buildTreeFromNodes(nodes []v1alpha1.ResourceNode, resourceStatuses []v1alpha1.ResourceStatus) []*TreeResource {
	resourceMap := make(map[string]*TreeResource)

	statusMap := make(map[string]*v1alpha1.ResourceStatus)
	for i := range resourceStatuses {
		status := &resourceStatuses[i]
		key := fmt.Sprintf("%s:%s:%s", status.Group, status.Kind, status.Name)
		if status.Namespace != "" {
			key = fmt.Sprintf("%s:%s:%s:%s", status.Group, status.Kind, status.Namespace, status.Name)
		}
		statusMap[key] = status
	}
	// Need this to avoid showing replicasets without pods
	// Check for having child refs doesnt work, since there can be some meta resources mapped to replicaset
	hasPods := make(map[string]bool)
	for i := range nodes {
		n := &nodes[i]
		if n.ResourceRef.Kind == "Pod" {
			for _, parent := range n.ParentRefs {
				parentNode := findNodeByUID(nodes, parent.UID)
				if parentNode != nil && parentNode.ResourceRef.Kind == "ReplicaSet" {
					hasPods[parent.UID] = true
				}
			}
		}
	}

	for i := range nodes {
		n := &nodes[i]
		tr := &TreeResource{
			Kind:      n.ResourceRef.Kind,
			Name:      n.ResourceRef.Name,
			Namespace: n.ResourceRef.Namespace,
			Expanded:  false,
			Children:  []*TreeResource{},
		}

		if n.Health != nil {
			tr.Health = string(n.Health.Status)
		} else {
			tr.Health = "Unknown"
		}

		statusKey := fmt.Sprintf("%s:%s:%s", n.ResourceRef.Group, n.ResourceRef.Kind, n.ResourceRef.Name)
		if n.ResourceRef.Namespace != "" {
			statusKey = fmt.Sprintf("%s:%s:%s:%s", n.ResourceRef.Group, n.ResourceRef.Kind, n.ResourceRef.Namespace, n.ResourceRef.Name)
		}

		if status, found := statusMap[statusKey]; found {
			tr.SyncStatus = string(status.Status)
		} else {
			tr.SyncStatus = "Unknown"
		}

		resourceMap[n.UID] = tr
	}

	for i := range nodes {
		n := &nodes[i]
		childTR, exists := resourceMap[n.UID]
		if !exists {
			continue
		}

		for _, pref := range n.ParentRefs {
			parentNode := findNodeByUID(nodes, pref.UID)
			parentTR, ok := resourceMap[pref.UID]

			if !ok {
				continue
			}

			if parentNode != nil && parentNode.ResourceRef.Kind == "Deployment" &&
				n.ResourceRef.Kind == "ReplicaSet" && !hasPods[n.UID] {
				continue
			}

			parentTR.Children = append(parentTR.Children, childTR)
		}
	}

	var roots []*TreeResource
	for i := range nodes {
		n := &nodes[i]
		resource, exists := resourceMap[n.UID]
		if !exists {
			continue
		}

		if len(n.ParentRefs) == 0 {
			roots = append(roots, resource)
		}
	}
	return roots
}

func findNodeByUID(nodes []v1alpha1.ResourceNode, uid string) *v1alpha1.ResourceNode {
	for i := range nodes {
		if nodes[i].UID == uid {
			return &nodes[i]
		}
	}
	return nil
}

func (s *ScreenAppResourcesList) applyFilters() {
	filteredResources := s.rootResources

	if s.kindFilter != "" {
		var filtered []*TreeResource
		for _, resource := range filteredResources {
			if resource.Kind == s.kindFilter {
				filtered = append(filtered, resource)
			}
		}
		filteredResources = filtered
	}

	if s.healthFilter != "" {
		var filtered []*TreeResource
		for _, resource := range filteredResources {
			if strings.EqualFold(resource.Health, s.healthFilter) {
				filtered = append(filtered, resource)
			}
		}
		filteredResources = filtered
	}

	if s.syncFilter != "" {
		var filtered []*TreeResource
		for _, resource := range filteredResources {
			if strings.EqualFold(resource.SyncStatus, s.syncFilter) {
				filtered = append(filtered, resource)
			}
		}
		filteredResources = filtered
	}

	s.filteredResources = filteredResources
	s.tableView.FillTableWithTree(s.filteredResources, s.getActiveFiltersText())
}

func (s *ScreenAppResourcesList) getActiveFiltersText() string {
	var parts []string

	if s.kindFilter != "" {
		parts = append(parts, fmt.Sprintf("Kind=%s", s.kindFilter))
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

func (s *ScreenAppResourcesList) showFilterMenu() {
	s.updateFilterCategories()

	filterCategories := []filters.FilterCategory{
		{
			Type:      filters.FilterTypeResourceKind,
			Title:     "Kind",
			Options:   s.filterCategories.kindList,
			Shortcuts: map[string]rune{},
		},
		{
			Type:      filters.FilterTypeHealth,
			Title:     "Health",
			Options:   s.filterCategories.healthList,
			Shortcuts: map[string]rune{},
		},
		{
			Type:      filters.FilterTypeSync,
			Title:     "Sync",
			Options:   s.filterCategories.syncList,
			Shortcuts: map[string]rune{},
		},
	}

	activeFilters := []filters.Filter{}

	filters.ShowFilterModal(
		s.app,
		s.pages,
		filterCategories,
		activeFilters,
		s.router,
		func(result filters.FilterModalResult) {
			if len(result.Filters) == 0 {
				s.kindFilter = ""
				s.healthFilter = ""
				s.syncFilter = ""
				s.applyFilters()
				return
			}
			if !result.Canceled {
				for _, filter := range result.Filters {
					switch filter.Type {
					case filters.FilterTypeResourceKind:
						s.kindFilter = filter.Value
					case filters.FilterTypeHealth:
						s.healthFilter = filter.Value
					case filters.FilterTypeSync:
						s.syncFilter = filter.Value
					}
				}
			}

			s.activeFilters = result.Filters
			s.applyFilters()
		},
	)
}

// Refactor. This func is a mess - too big
func (s *ScreenAppResourcesList) onTableKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'q':
		s.app.Stop()
		return nil
	case 'b':
		s.instanceInfo.ClearAppInfo()
		s.router.Back()
		return nil
	case 'T':
		s.toggleExpansionAll()
		return nil
	case 't':
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
				s.tableView.FillTableWithTree(s.visibleResources, s.getActiveFiltersText())
				s.footer.UpdateResourceCount(len(s.visibleResources))
			}
		}
		return nil
		//TODO: Search via child resources as well?
	case '/', ':':
		s.showSearchBar()
		return nil
	case 'F', 'f':
		s.showFilterMenu()
		return nil
	case 'd':
		if s.kindFilter == "Deployment" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "Deployment"
		}
		s.applyFilters()
		return nil
	case 'i':
		if s.kindFilter == "Ingress" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "Ingress"
		}
		s.applyFilters()
		return nil
	case 's':
		if s.kindFilter == "Service" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "Service"
		}
		s.applyFilters()
		return nil
	case 'c':
		if s.kindFilter == "ConfigMap" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "ConfigMap"
		}
		s.applyFilters()
		return nil
	case 'h':
		if s.healthFilter == "Healthy" {
			s.healthFilter = ""
		} else {
			s.healthFilter = "Healthy"
		}
		s.applyFilters()
		return nil
	case 'o':
		if s.syncFilter == "OutOfSync" {
			s.syncFilter = ""
		} else {
			s.syncFilter = "OutOfSync"
		}
		s.applyFilters()
		return nil
	case 'p':
		if s.healthFilter == "Progressing" {
			s.healthFilter = ""
		} else {
			s.healthFilter = "Progressing"
		}
		s.applyFilters()
		return nil
	case '?':
		s.pages.SwitchToPage("help")
		s.app.SetFocus(s.helpView.Grid)
	}
	if event.Key() == tcell.KeyEnter {
		row, _ := s.table.GetSelection()
		if row > 0 && row-1 < len(s.visibleResources) {
			selected := s.visibleResources[row-1]
			nodeKey := getNodeKey(selected)

			if originalNode, exists := s.originalNodes[nodeKey]; exists {
				if len(originalNode.Children) > 0 {
					if !originalNode.Expanded {
						expand(originalNode, 0)
					} else {
						collapse(originalNode, 0)
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
	s.tableView.FillTableWithTree(s.visibleResources, s.getActiveFiltersText())
	s.footer.UpdateResourceCount(len(s.visibleResources))
}

func (s *ScreenAppResourcesList) showToast(message string, duration time.Duration) {
	var toast *components.SimpleSearchBar
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, 1, -1, 1) // topBar, toast, table, foots.er
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

func (s *ScreenAppResourcesList) updateFilterCategories() {
	kinds := make(map[string]bool)
	healthStatuses := make(map[string]bool)
	syncStatuses := make(map[string]bool)

	for _, resource := range s.rootResources {
		kinds[resource.Kind] = true
		healthStatuses[resource.Health] = true
		syncStatuses[resource.SyncStatus] = true
	}
	s.filterCategories.kindList = mapKeysToSortedSlice(kinds)
	s.filterCategories.healthList = mapKeysToSortedSlice(healthStatuses)
	s.filterCategories.syncList = mapKeysToSortedSlice(syncStatuses)

}

func mapKeysToSortedSlice(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for key := range m {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func (s *ScreenAppResourcesList) Name() string {
	return fmt.Sprintf("ApplicationResourcesList-%s", s.selectedApp.Name)
}
