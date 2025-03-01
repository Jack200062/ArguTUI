package applicationResourcesList

import (
	"fmt"
	"strings"
	"time"

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
	treeMode         bool
	selectedAppName  string
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
		treeMode:        true,
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

func flattenResources(resources []*TreeResource, depth int) []*TreeResource {
	var result []*TreeResource
	for _, r := range resources {
		r.Depth = depth
		result = append(result, r)
		if r.Expanded {
			result = append(result, flattenResources(r.Children, depth+1)...)
		}
	}
	return result
}

func flattenAllResources(resources []*TreeResource, depth int) []*TreeResource {
	var result []*TreeResource
	for _, r := range resources {
		copyNode := *r
		copyNode.Depth = depth
		result = append(result, &copyNode)
		result = append(result, flattenAllResources(r.Children, depth+1)...)
	}
	return result
}

func (s *ScreenAppResourcesList) Init() tview.Primitive {
	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String()).
		SetTextAlign(tview.AlignLeft)
	instanceBox.SetBorder(false)
	shortCutInfo := components.ShortcutBar()

	topBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(instanceBox, 0, 1, false).
		AddItem(shortCutInfo, 0, 1, false)

	s.searchBar = components.NewSimpleSearchBar("Search: ", 0)
	s.searchBar.InputField.SetDoneFunc(s.searchDone)

	// Move to components.SimpleSearchBar
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
		s.fillTableNormalMode()
	} else {
		s.fillTableTreeMode()
	}

	return s.pages
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
		s.treeMode = true
		if err := s.buildTreeFromResourceTree(); err == nil {
			for _, root := range s.rootResources {
				expandFully(root)
			}
			s.fillTableTreeMode()
		}
		return nil
	case '/', ':':
		s.showSearchBar()
		return nil
	}

	if event.Key() == tcell.KeyEnter && s.treeMode {
		row, _ := s.table.GetSelection()
		if row > 0 && row-1 < len(s.visibleResources) {
			selected := s.visibleResources[row-1]
			if !selected.Expanded {
				expandFully(selected)
			} else {
				collapseFully(selected)
			}
			s.fillTableTreeMode()
		}
		return nil
	}

	return event
}

func (s *ScreenAppResourcesList) showSearchBar() {
	s.fillTableTreeMode()
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

	if s.treeMode {
		_ = s.buildTreeFromResourceTree()
		allNodes := flattenAllResources(s.rootResources, 0)
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
			row++
		}
		return
	}

	var fr []argocd.Resource
	for _, r := range s.resources {
		text := strings.ToLower(r.Kind + " " + r.Name + " " + r.Namespace + " " + r.HealthStatus + " " + r.SyncStatus)
		if strings.Contains(text, lq) {
			fr = append(fr, r)
		}
	}
	s.filteredResults = fr
	s.fillTableNormalMode()
}

func (s *ScreenAppResourcesList) fillTableNormalMode() {
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
	for _, r := range s.filteredResults {
		s.table.SetCell(row, 0, tview.NewTableCell(r.Kind).SetExpansion(1))
		s.table.SetCell(row, 1, tview.NewTableCell(r.Name).SetExpansion(1))
		s.table.SetCell(row, 2, tview.NewTableCell(r.HealthStatus).SetExpansion(1))
		s.table.SetCell(row, 3, tview.NewTableCell(r.SyncStatus).SetExpansion(1))
		s.table.SetCell(row, 4, tview.NewTableCell(r.Namespace).SetExpansion(1))
		row++
	}
}

func (s *ScreenAppResourcesList) buildTreeFromResourceTree() error {
	appTree, err := s.client.GetResourceTree(s.selectedAppName)
	if err != nil {
		return err
	}
	s.rootResources = buildTreeFromNodes(appTree.Nodes)
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
	s.visibleResources = flattenResources(s.rootResources, 0)
	row := 1
	for _, tr := range s.visibleResources {
		indent := strings.Repeat("  ", tr.Depth)
		var marker string
		if len(tr.Children) > 0 {
			if tr.Expanded {
				marker = "▼ "
			} else {
				marker = "▶ "
			}
		} else {
			marker = "  "
		}
		kindText := indent + marker + tr.Kind
		s.table.SetCell(row, 0, tview.NewTableCell(kindText).SetExpansion(1))
		s.table.SetCell(row, 1, tview.NewTableCell(tr.Name).SetExpansion(1))
		s.table.SetCell(row, 2, tview.NewTableCell(tr.Health).SetExpansion(1))
		s.table.SetCell(row, 3, tview.NewTableCell(tr.SyncStatus).SetExpansion(1))
		s.table.SetCell(row, 4, tview.NewTableCell(tr.Namespace).SetExpansion(1))
		row++
	}
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
