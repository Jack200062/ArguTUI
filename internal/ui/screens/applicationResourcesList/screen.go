package applicationResourcesList

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

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
	// Cached lower-cased concatenation for search
	SearchIndex string
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
	cachedFlattened  []*TreeResource
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

	// unique screen name per application
	name string
}

func New(
	app *tview.Application,
	resources []argocd.Resource,
	selectedAppName string,
	appHealthStatus string,
	appSyncStatus string,
	r *ui.Router,
	instanceInfo *common.InstanceInfo,
	client *argocd.ArgoCdClient,
) *ScreenAppResourcesList {
	instanceInfo = instanceInfo.WithAppInfo(selectedAppName, appHealthStatus, appSyncStatus)

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
		appHealthStatus: appHealthStatus,
		appSyncStatus:   appSyncStatus,
		name:            "ApplicationResourcesList:" + selectedAppName,
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

	s.visibleResources = s.cachedFlattened

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

	if kindFilter != "" {
		s.filterResourcesByKind(kindFilter)
	} else if healthFilter != "" || syncFilter != "" {
		s.filterResourcesByStatus(healthFilter, syncFilter)
	} else {
		s.visibleResources = s.cachedFlattened
	}

	s.fillTableTreeMode()
}

func (s *ScreenAppResourcesList) filterResourcesByKind(kindFilter string) {
	if kindFilter == "" {
		s.visibleResources = s.cachedFlattened
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
		s.visibleResources = s.cachedFlattened
	}
}

func (s *ScreenAppResourcesList) filterResourcesByStatus(healthFilter, syncFilter string) {
	if healthFilter == "" && syncFilter == "" {
		s.cachedFlattened = flattenResourcesWithLines(s.rootResources, 0, nil)
		s.visibleResources = s.cachedFlattened
		return
	}

	allNodes := s.cachedFlattened
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

	s.cachedFlattened = flattenResourcesWithLines(s.rootResources, 0, nil)
	s.visibleResources = s.cachedFlattened
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
		// Без сети: восстановить видимый список из уже загруженного дерева
		s.visibleResources = s.cachedFlattened
		s.fillTableTreeMode()
		return
	}

	lq := strings.ToLower(query)

	// Используем уже имеющееся дерево вместо повторного сетевого запроса
	allNodes := s.cachedFlattened
	var filtered []*TreeResource
	for _, n := range allNodes {
		// Используем предрасчитанный индекс
		if strings.Contains(n.SearchIndex, lq) {
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
	// Обновить кэш развёрнутого списка
	s.cachedFlattened = flattenResourcesWithLines(s.rootResources, 0, nil)
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
		// Precompute search index
		tr.SearchIndex = strings.ToLower(tr.Kind + " " + tr.Name + " " + tr.Namespace + " " + tr.Health + " " + tr.SyncStatus)
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
	// esc -> back like k9s
	if event.Key() == tcell.KeyEsc {
		s.router.Back()
		return nil
	}
	// k9s-like navigation keys
	switch event.Key() {
	case tcell.KeyCtrlU:
		s.moveSelection(-s.halfPage())
		return nil
	case tcell.KeyCtrlD:
		s.moveSelection(s.halfPage())
		return nil
	case tcell.KeyCtrlB, tcell.KeyPgUp:
		s.moveSelection(-s.fullPage())
		return nil
	case tcell.KeyCtrlF, tcell.KeyPgDn:
		s.moveSelection(s.fullPage())
		return nil
	}
	switch event.Rune() {
	case 'j':
		s.moveSelection(1)
		return nil
	case 'k':
		s.moveSelection(-1)
		return nil
	case 'g':
		s.gotoTop()
		return nil
	case 'G':
		s.gotoBottom()
		return nil
	case 't':
		s.toggleExpansionAll()
		return nil
	case '/', ':':
		s.showSearchBar()
		return nil
	case 'y':
		yamlText := s.selectedResourceYAML(false)
		if yamlText == "" {
			s.showToast("YAML not available", 2*time.Second)
			return nil
		}
		modal := components.TextModal("YAML", yamlText, func() { s.app.SetRoot(s.pages, true) })
		s.app.SetRoot(modal, true)
		return nil
	case 'e':
		// Open in external editor ($KUBE_EDITOR/$EDITOR or vim)
		s.promptEditResource()
		return nil
	case '=':
		if desired, live, ok := s.selectedResourcePair(); ok {
			// normalize both sides to YAML for consistent diff rendering
			left := s.normalizeToYAML(desired)
			right := s.normalizeToYAML(live)
			view := components.NewSideBySideDiff("Diff (desired vs live)", " Desired ", " Live ", left, right, func() { s.app.SetRoot(s.pages, true) })
			s.app.SetRoot(view, true)
		} else {
			s.showToast("Diff not available", 2*time.Second)
		}
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

					s.cachedFlattened = flattenResourcesWithLines(s.rootResources, 0, nil)
					s.visibleResources = s.cachedFlattened
					s.fillTableTreeMode()
				}
			}
		}
		return nil
	}

	return event
}

// Navigation helpers for k9s-like keys
func (s *ScreenAppResourcesList) moveSelection(delta int) {
	row, _ := s.table.GetSelection()
	row += delta
	if row < 1 {
		row = 1
	}
	if row > len(s.visibleResources) {
		row = len(s.visibleResources)
	}
	if row >= 1 {
		s.table.Select(row, 0)
	}
}

func (s *ScreenAppResourcesList) halfPage() int { return 10 }
func (s *ScreenAppResourcesList) fullPage() int { return 20 }
func (s *ScreenAppResourcesList) gotoTop() {
	if len(s.visibleResources) > 0 {
		s.table.Select(1, 0)
	}
}
func (s *ScreenAppResourcesList) gotoBottom() {
	if n := len(s.visibleResources); n > 0 {
		s.table.Select(n, 0)
	}
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

// Helpers to fetch YAML/diff for current selection
func (s *ScreenAppResourcesList) selectedResourcePair() (desired string, live string, ok bool) {
	row, _ := s.table.GetSelection()
	if row <= 0 || row-1 >= len(s.visibleResources) {
		return "", "", false
	}
	r := s.visibleResources[row-1]
	d, l, err := s.client.GetResourceManifest(s.selectedAppName, "", r.Kind, r.Namespace, r.Name)
	if err != nil {
		return "", "", false
	}
	return d, l, true
}

func (s *ScreenAppResourcesList) selectedResourceYAML(preferDesired bool) string {
	d, l, ok := s.selectedResourcePair()
	if !ok {
		return ""
	}
	// prefer live by default for readability
	raw := l
	if preferDesired && d != "" {
		raw = d
	} else if raw == "" {
		raw = d
	}
	// Normalize and sanitize to YAML
	return s.normalizeToYAML(raw)
}

// normalizeToYAML tries to convert JSON to YAML, otherwise returns as-is
func (s *ScreenAppResourcesList) normalizeToYAML(text string) string {
	var any interface{}
	// Try YAML first
	if err := yaml.Unmarshal([]byte(text), &any); err == nil {
		cleaned := sanitizeK8sFields(any)
		if y, err := yaml.Marshal(cleaned); err == nil {
			return string(y)
		}
	}
	// Fallback to JSON parsing
	if err := json.Unmarshal([]byte(text), &any); err == nil {
		cleaned := sanitizeK8sFields(any)
		if y, err := yaml.Marshal(cleaned); err == nil {
			return string(y)
		}
		if buf, err := json.MarshalIndent(cleaned, "", "  "); err == nil {
			return string(buf)
		}
	}
	return text
}

// promptEditResource runs `kubectl edit` for the selected resource in the user's $EDITOR
func (s *ScreenAppResourcesList) promptEditResource() {
	row, _ := s.table.GetSelection()
	if row <= 0 || row-1 >= len(s.visibleResources) {
		s.showToast("Nothing selected", 2*time.Second)
		return
	}

	// Возьмём актуальный YAML (Live) и дадим пользователю отредактировать его во внешнем редакторе
	yamlText := s.selectedResourceYAML(false)
	if strings.TrimSpace(yamlText) == "" {
		s.showToast("YAML not available", 2*time.Second)
		return
	}

	tmp, err := os.CreateTemp("", "argutui-edit-*.yaml")
	if err != nil {
		s.showToast("tmpfile error", 3*time.Second)
		return
	}
	tmpPath := tmp.Name()
	_, _ = tmp.WriteString(yamlText)
	_ = tmp.Close()

	// Выбираем редактор: KUBE_EDITOR > EDITOR > vim
	editor := os.Getenv("KUBE_EDITOR")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vim"
	}

	// Запускаем редактор синхронно в suspend-режиме
	s.app.Suspend(func() {
		cmd := exec.Command(editor, tmpPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
	})

	// После выхода из редактора применим изменения
	go func() {
		defer os.Remove(tmpPath)
		cmd := exec.Command("kubectl", "apply", "-f", tmpPath)
		out, err := cmd.CombinedOutput()
		s.app.QueueUpdateDraw(func() {
			if err != nil {
				s.showToast("Apply failed: "+string(out), 5*time.Second)
			} else {
				s.showToast("Applied", 2*time.Second)
			}
		})
	}()
}

// sanitizeK8sFields removes noisy, server-populated fields to make diffs cleaner
func sanitizeK8sFields(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		// Remove status entirely (server-populated)
		delete(val, "status")
		// If metadata exists, clean it
		if metaRaw, ok := val["metadata"]; ok {
			if meta, ok := metaRaw.(map[string]interface{}); ok {
				delete(meta, "managedFields")
				delete(meta, "resourceVersion")
				delete(meta, "selfLink")
				delete(meta, "uid")
				delete(meta, "creationTimestamp")
				delete(meta, "deletionTimestamp")
				// Clean annotations
				if annRaw, ok := meta["annotations"]; ok {
					if ann, ok := annRaw.(map[string]interface{}); ok {
						// Known noisy annotations
						noisy := map[string]bool{
							"kubectl.kubernetes.io/last-applied-configuration": true,
							"kubectl.kubernetes.io/restartedAt":                true,
							"deployment.kubernetes.io/revision":                true,
						}
						for k := range noisy {
							delete(ann, k)
						}
						// If annotations becomes empty, drop it
						if len(ann) == 0 {
							delete(meta, "annotations")
						}
					}
				}
				// If metadata becomes empty, drop it
				if len(meta) == 0 {
					delete(val, "metadata")
				} else {
					val["metadata"] = meta
				}
			}
		}
		// Recurse into all nested maps/arrays
		for k, child := range val {
			val[k] = sanitizeK8sFields(child)
		}
		return val
	case []interface{}:
		for i, child := range val {
			val[i] = sanitizeK8sFields(child)
		}
		return val
	default:
		return v
	}
}

func (s *ScreenAppResourcesList) renderUnifiedDiff(desired, live string) string {
	if desired == live {
		return "No changes"
	}
	// Normalize to YAML if possible for consistent diff
	toYAML := func(s string) string {
		var v interface{}
		if json.Unmarshal([]byte(s), &v) == nil {
			if y, err := yaml.Marshal(v); err == nil {
				return string(y)
			}
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		}
		return s
	}
	desired = toYAML(desired)
	live = toYAML(live)
	// argocd-like unified diff markers with color
	dl := strings.Split(desired, "\n")
	ll := strings.Split(live, "\n")
	var out []string
	out = append(out, " Desired ", " Live ")
	i, j := 0, 0
	for i < len(dl) || j < len(ll) {
		if i < len(dl) && j < len(ll) {
			if dl[i] == ll[j] {
				out = append(out, " "+dl[i])
				i++
				j++
			} else {
				out = append(out, "[#ff6b6b]-"+dl[i]+"[-]")
				out = append(out, "[#51cf66]+"+ll[j]+"[-]")
				i++
				j++
			}
		} else if i < len(dl) {
			out = append(out, "[#ff6b6b]-"+dl[i]+"[-]")
			i++
		} else if j < len(ll) {
			out = append(out, "[#51cf66]+"+ll[j]+"[-]")
			j++
		}
	}
	return strings.Join(out, "\n")
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
	if s.name != "" {
		return s.name
	}
	return "ApplicationResourcesList"
}
