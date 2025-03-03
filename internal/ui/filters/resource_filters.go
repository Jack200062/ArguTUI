package filters

import (
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	ResourceKindFilter FilterType = "resourceKind"
)

type ResourceFilterManager struct {
	app            *tview.Application
	pages          *tview.Pages
	kindTypes      map[string]bool
	namespaces     map[string]bool
	kinds          []string
	healthStatuses []string
	syncStatuses   []string

	Filters []FilterState

	OnFilterChanged func([]FilterState)
}

func NewResourceFilterManager(app *tview.Application, pages *tview.Pages) *ResourceFilterManager {
	return &ResourceFilterManager{
		app:            app,
		pages:          pages,
		kindTypes:      make(map[string]bool),
		namespaces:     make(map[string]bool),
		healthStatuses: []string{"Healthy", "Progressing", "Degraded", "Suspended", "Unknown"},
		syncStatuses:   []string{"Synced", "OutOfSync"},
		Filters:        []FilterState{},
	}
}

func (f *ResourceFilterManager) SetHealthStatuses(statuses []string) {
	if len(statuses) > 0 {
		f.healthStatuses = statuses
	}
}

func (f *ResourceFilterManager) SetSyncStatuses(statuses []string) {
	if len(statuses) > 0 {
		f.syncStatuses = statuses
	}
}

func (f *ResourceFilterManager) SetFilterChangedHandler(handler func([]FilterState)) {
	f.OnFilterChanged = handler
}

func (f *ResourceFilterManager) ExtractKindsFromResources(kinds map[string]bool) {
	f.kindTypes = kinds

	f.kinds = make([]string, 0, len(f.kindTypes))
	for kind := range f.kindTypes {
		f.kinds = append(f.kinds, kind)
	}
	sort.Strings(f.kinds)
}

func (f *ResourceFilterManager) ShowFilterMenu() {
	categories := []FilterCategory{}

	if len(f.kinds) > 0 {
		categories = append(categories, FilterCategory{
			Title:     "Kind",
			Type:      ResourceKindFilter,
			Options:   f.kinds,
			Shortcuts: f.getKindShortcuts(),
		})
	}

	if len(f.healthStatuses) > 0 {
		categories = append(categories, FilterCategory{
			Title:     "Health Status",
			Type:      HealthFilter,
			Options:   f.healthStatuses,
			Shortcuts: StandardHealthShortcuts(),
		})
	}

	if len(f.syncStatuses) > 0 {
		categories = append(categories, FilterCategory{
			Title:     "Sync Status",
			Type:      SyncFilter,
			Options:   f.syncStatuses,
			Shortcuts: StandardSyncShortcuts(),
		})
	}

	if len(categories) > 0 {
		ShowMultiFilterMenu(
			f.app,
			MultiFilterOptions{
				Title:         "FILTER RESOURCES",
				ActiveFilters: f.Filters,
				Categories:    categories,
				AsOverlay:     true,
				Position: OverlayPosition{
					Top:    3,
					Left:   10,
					Right:  10,
					Bottom: 5,
				},
			},
			f.pages,
			func(result FilterResult) {
				if !result.Canceled {
					f.Filters = result.Filters
					f.applyFilters()
				}
			},
		)
	} else {
		modal := tview.NewModal().
			SetText("No filter categories available").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				f.app.SetRoot(f.pages, true)
			})
		f.app.SetRoot(modal, true)
	}
}

func (f *ResourceFilterManager) GetActiveFiltersText() string {
	return GetActiveFiltersText(f.Filters)
}

func (f *ResourceFilterManager) SetFilter(filterType FilterType, value string) {
	f.Filters = UpdateFilter(f.Filters, filterType, value)
	f.applyFilters()
}

func (f *ResourceFilterManager) GetFilter(filterType FilterType) string {
	value, found := FindFilterByType(f.Filters, filterType)
	if found {
		return value
	}
	return ""
}

func (f *ResourceFilterManager) ClearFilters() {
	f.Filters = []FilterState{}
	f.applyFilters()
}

func (f *ResourceFilterManager) ToggleFilter(filterType FilterType, value string) {
	currentValue, found := FindFilterByType(f.Filters, filterType)
	if found && currentValue == value {
		f.SetFilter(filterType, "")
	} else {
		f.SetFilter(filterType, value)
	}
}

func (f *ResourceFilterManager) applyFilters() {
	if f.OnFilterChanged != nil {
		f.OnFilterChanged(f.Filters)
	}
}

func (f *ResourceFilterManager) getKindShortcuts() map[string]rune {
	standardShortcuts := map[string]rune{
		"Deployment":  'd',
		"Service":     's',
		"Ingress":     'i',
		"ConfigMap":   'c',
		"Secret":      'x',
		"Pod":         'p',
		"Job":         'j',
		"CronJob":     'r',
		"Namespace":   'n',
		"StatefulSet": 't',
		"DaemonSet":   'a',
		"ReplicaSet":  'e',
	}

	shortcuts := make(map[string]rune)
	for kind, shortcut := range standardShortcuts {
		if f.kindTypes[kind] {
			shortcuts[kind] = shortcut
		}
	}

	return shortcuts
}

func StandardResourceKindShortcuts() map[string]rune {
	return map[string]rune{
		"Deployment":  'd',
		"Service":     's',
		"Ingress":     'i',
		"ConfigMap":   'c',
		"Secret":      'x',
		"Pod":         'p',
		"Job":         'j',
		"CronJob":     'r',
		"Namespace":   'n',
		"StatefulSet": 't',
		"DaemonSet":   'a',
		"ReplicaSet":  'e',
	}
}

func GetResourceKindColor(kind string) tcell.Color {
	return tcell.ColorWhite
}
