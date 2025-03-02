package applicationResourcesList

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
	kindFilter       string
	allExpanded      bool

	originalNodes map[string]*TreeResource
}

func New(
	app *tview.Application,
	resources []argocd.Resource,
	selectedAppName string,
	r *ui.Router,
	instanceInfo *common.InstanceInfo,
	client *argocd.ArgoCdClient,
) *ScreenAppResourcesList {
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

func (s *ScreenAppResourcesList) Init() tview.Primitive {
	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String()).
		SetTextAlign(tview.AlignLeft)
	instanceBox.SetBorder(false)

	shortCutInfo := tview.NewTextView().
		SetText(" q Quit ? Help f Filter \n d Deploy s Service c Config t Toggle").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorYellow)

	topBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceBox, 0, 1, false).
		AddItem(shortCutInfo, 0, 1, false)

	s.searchBar = components.NewSimpleSearchBar("Search: ", 0)
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

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

	s.table = tview.NewTable().
		SetSelectable(true, false).
		SetBorders(false)
	s.table.SetBorder(true)
	s.table.SetTitle(fmt.Sprintf(" Resources for %s ", s.selectedAppName))

	s.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(0).
		SetBorders(true)
	s.grid.AddItem(topBar, 0, 0, 1, 1, 0, 0, false)
	s.grid.AddItem(s.table, 1, 0, 1, 1, 0, 0, true)

	s.pages = tview.NewPages().
		AddPage("main", s.grid, true, true)

	s.table.SetInputCapture(s.onTableKey)

	if err := s.buildTreeFromResourceTree(); err != nil {
		s.showToast(fmt.Sprintf("Error building tree: %v", err), 3*time.Second)
	}

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
	s.fillTableTreeMode()
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
	case 'f', 'F':
		s.showKindFilterMenu()
		return nil
	case 'd', 'D':
		if s.kindFilter == "Deployment" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "Deployment"
		}
		s.fillTableTreeMode()
		return nil
	case 's', 'S':
		if s.kindFilter == "Service" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "Service"
		}
		s.fillTableTreeMode()
		return nil
	case 'i', 'I':
		if s.kindFilter == "Ingress" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "Ingress"
		}
		s.fillTableTreeMode()
		return nil
	case 'c', 'C':
		if s.kindFilter == "ConfigMap" {
			s.kindFilter = ""
		} else {
			s.kindFilter = "ConfigMap"
		}
		s.fillTableTreeMode()
		return nil
	}

	if event.Key() == tcell.KeyEnter {
		row, _ := s.table.GetSelection()
		if row > 0 && row-1 < len(s.visibleResources) {
			selected := s.visibleResources[row-1]
			nodeKey := getNodeKey(selected)
			if originalNode, exists := s.originalNodes[nodeKey]; exists {
				if !originalNode.Expanded {
					expandFully(originalNode)
				} else {
					collapseFully(originalNode)
				}
				s.fillTableTreeMode()
			}
		}
		return nil
	}

	return event
}

func generateTreePrefix(node *TreeResource, parentLineInfo *TreeLineInfo) string {
	if node.Depth == 0 {
		if len(node.Children) > 0 {
			if node.Expanded {
				return "▼ "
			}
			return "▶ "
		}
		return "  "
	}

	var prefix strings.Builder

	for i := 0; i < node.Depth-1; i++ {
		if parentLineInfo != nil && i < len(parentLineInfo.LineChars) {
			if parentLineInfo.LineChars[i] == '│' {
				prefix.WriteString("│ ")
			} else {
				prefix.WriteString("  ")
			}
		} else {
			prefix.WriteString("  ")
		}
	}

	if node.IsLast {
		prefix.WriteString("└─")
	} else {
		prefix.WriteString("├─")
	}

	if len(node.Children) > 0 {
		if node.Expanded {
			prefix.WriteString("▼ ")
		} else {
			prefix.WriteString("▶ ")
		}
	} else {
		prefix.WriteString("  ")
	}

	return prefix.String()
}

func (s *ScreenAppResourcesList) showKindFilterMenu() {
	rootKindsWithChildren := make(map[string]bool)

	var findRootKindsWithChildren func([]*TreeResource, bool)
	findRootKindsWithChildren = func(nodes []*TreeResource, isRoot bool) {
		for _, node := range nodes {
			if isRoot && len(node.Children) > 0 {
				rootKindsWithChildren[node.Kind] = true
			}
			findRootKindsWithChildren(node.Children, false)
		}
	}

	findRootKindsWithChildren(s.rootResources, true)

	sortedKinds := make([]string, 0, len(rootKindsWithChildren))
	for kind := range rootKindsWithChildren {
		sortedKinds = append(sortedKinds, kind)
	}
	sort.Strings(sortedKinds)

	if len(sortedKinds) == 0 {
		s.showToast("No root resources with children found", 2*time.Second)
		return
	}

	standardShortcuts := map[string]rune{
		"Deployment": 'd',
		"Service":    's',
		"Ingress":    'i',
		"ConfigMap":  'c',
		"Secret":     'x',
		"Pod":        'p',
		"Job":        'j',
		"CronJob":    'r',
		"Namespace":  'n',
	}

	var shortcutsInfo strings.Builder
	shortcutsInfo.WriteString("Keyboard shortcuts: [a] All ")

	for _, kind := range sortedKinds {
		shortcut, hasShortcut := standardShortcuts[kind]
		if hasShortcut {
			shortcutsInfo.WriteString(fmt.Sprintf("[%c] %s ", shortcut, kind))
		}
	}

	modal := tview.NewModal()
	modalText := "Choose parent resource type to filter\n\n" + shortcutsInfo.String()
	modal.SetText(modalText)

	buttons := []string{"All (clear filter)"}
	for _, kind := range sortedKinds {
		buttons = append(buttons, kind)
	}

	modal.SetBackgroundColor(tcell.ColorDarkBlue)
	modal.AddButtons(buttons)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 {
			s.kindFilter = ""
		} else if buttonIndex > 0 && buttonIndex <= len(sortedKinds) {
			s.kindFilter = sortedKinds[buttonIndex-1]
		}
		s.fillTableTreeMode()
		s.app.SetRoot(s.pages, true)
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			s.app.SetRoot(s.pages, true)
			return nil
		}

		switch event.Rune() {
		case 'a', 'A':
			s.kindFilter = ""
			s.fillTableTreeMode()
			s.app.SetRoot(s.pages, true)
			return nil
		default:
			for i, kind := range sortedKinds {
				shortcut, hasShortcut := standardShortcuts[kind]
				if hasShortcut && (event.Rune() == shortcut || event.Rune() == unicode.ToUpper(shortcut)) {
					s.kindFilter = sortedKinds[i]
					s.fillTableTreeMode()
					s.app.SetRoot(s.pages, true)
					return nil
				}
			}
		}

		return event
	})

	s.app.SetRoot(modal, true)
}

func (s *ScreenAppResourcesList) showSearchBar() {
	s.grid.RemoveItem(s.table)
	s.searchBar.InputField.SetText("")

	s.grid.SetRows(3, 1, -1)
	s.grid.AddItem(s.searchBar.InputField, 1, 0, 1, 1, 0, 0, false)
	s.grid.AddItem(s.table, 2, 0, 1, 1, 0, 0, true)
	s.app.SetFocus(s.searchBar.InputField)
}

func (s *ScreenAppResourcesList) hideSearchBar() {
	s.grid.RemoveItem(s.searchBar.InputField)
	s.grid.RemoveItem(s.table)
	s.grid.SetRows(3, 0)
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

	s.table.Clear()
	headers := []string{"Kind", "Name", "Health", "SyncStatus", "Namespace"}
	for col, h := range headers {
		cell := tview.NewTableCell("[::b]" + h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, cell)
	}

	row := 1
	for _, tr := range filtered {
		s.table.SetCell(row, 0, tview.NewTableCell(tr.Kind).SetExpansion(1))
		s.table.SetCell(row, 1, tview.NewTableCell(tr.Name).SetExpansion(1))
		s.table.SetCell(row, 2, tview.NewTableCell(tr.Health).SetExpansion(1))
		s.table.SetCell(row, 3, tview.NewTableCell(tr.SyncStatus).SetExpansion(1))
		s.table.SetCell(row, 4, tview.NewTableCell(tr.Namespace).SetExpansion(1))

		rowColor := common.RowColorForStatuses(tr.Health, tr.SyncStatus)
		common.SetRowColor(s.table, row, len(headers), rowColor)

		row++
	}
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
			Expanded:  false,
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

func (s *ScreenAppResourcesList) fillTableTreeMode() {
	s.table.Clear()
	headers := []string{"Kind", "Name", "Health", "SyncStatus", "Namespace"}
	for col, h := range headers {
		cell := tview.NewTableCell("[::b]" + h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		s.table.SetCell(0, col, cell)
	}

	if s.kindFilter != "" {
		s.table.SetTitle(fmt.Sprintf(" Resources for %s (Filtered: %s) ", s.selectedAppName, s.kindFilter))
	} else {
		s.table.SetTitle(fmt.Sprintf(" Resources for %s ", s.selectedAppName))
	}

	var rootsToShow []*TreeResource

	if s.kindFilter == "" {
		rootsToShow = s.rootResources
	} else {
		rootsToShow = s.filterResourcesWithChildren()
	}

	s.visibleResources = flattenResourcesWithLines(rootsToShow, 0, nil)

	row := 1
	for _, tr := range s.visibleResources {
		var lineInfo *TreeLineInfo

		treePrefix := generateTreePrefix(tr, lineInfo)
		kindText := treePrefix + tr.Kind

		s.table.SetCell(row, 0, tview.NewTableCell(kindText).SetExpansion(1))
		s.table.SetCell(row, 1, tview.NewTableCell(tr.Name).SetExpansion(1))
		s.table.SetCell(row, 2, tview.NewTableCell(tr.Health).SetExpansion(1))
		s.table.SetCell(row, 3, tview.NewTableCell(tr.SyncStatus).SetExpansion(1))
		s.table.SetCell(row, 4, tview.NewTableCell(tr.Namespace).SetExpansion(1))

		rowColor := common.RowColorForStatuses(tr.Health, tr.SyncStatus)
		common.SetRowColor(s.table, row, len(headers), rowColor)

		row++
	}
}

func (s *ScreenAppResourcesList) filterResourcesWithChildren() []*TreeResource {
	if s.kindFilter == "" {
		return s.rootResources
	}

	var filteredRoots []*TreeResource

	var filterNode func(*TreeResource, bool) *TreeResource
	filterNode = func(node *TreeResource, isChildOfTarget bool) *TreeResource {
		if node.Kind == s.kindFilter || isChildOfTarget {
			nodeCopy := *node

			var filteredChildren []*TreeResource
			for _, child := range node.Children {
				childIsChildOfTarget := isChildOfTarget || node.Kind == s.kindFilter
				filteredChild := filterNode(child, childIsChildOfTarget)
				if filteredChild != nil {
					filteredChildren = append(filteredChildren, filteredChild)
				}
			}

			nodeCopy.Children = filteredChildren
			return &nodeCopy
		}

		var filteredChildren []*TreeResource
		for _, child := range node.Children {
			filteredChild := filterNode(child, false)
			if filteredChild != nil {
				filteredChildren = append(filteredChildren, filteredChild)
			}
		}

		if len(filteredChildren) > 0 {
			nodeCopy := *node
			nodeCopy.Children = filteredChildren
			return &nodeCopy
		}

		return nil
	}

	for _, root := range s.rootResources {
		filteredRoot := filterNode(root, false)
		if filteredRoot != nil {
			filteredRoots = append(filteredRoots, filteredRoot)
		}
	}

	return filteredRoots
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

func (s *ScreenAppResourcesList) showToast(message string, duration time.Duration) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			s.app.SetRoot(s.pages, true)
		})
	s.app.SetRoot(modal, true)
	go func() {
		time.Sleep(duration)
		s.app.QueueUpdateDraw(func() {
			s.app.SetRoot(s.pages, true)
		})
	}()
}

func (s *ScreenAppResourcesList) Name() string {
	return "ApplicationResourcesList"
}
