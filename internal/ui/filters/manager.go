package filters

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type FilterManager struct {
	app   *tview.Application
	pages *tview.Pages

	Categories    []FilterCategory
	ActiveFilters []Filter

	OnFilterChanged func([]Filter)
}

func NewFilterManager(app *tview.Application, pages *tview.Pages) *FilterManager {
	return &FilterManager{
		app:           app,
		pages:         pages,
		Categories:    []FilterCategory{},
		ActiveFilters: []Filter{},
	}
}

func (fm *FilterManager) SetCategories(categories []FilterCategory) {
	fm.Categories = categories
}

func (fm *FilterManager) AddCategory(category FilterCategory) {
	fm.Categories = append(fm.Categories, category)
}

func (fm *FilterManager) ApplyFilter(filterType string, value string) {
	if value == "" {
		for i, filter := range fm.ActiveFilters {
			if filter.Type == filterType {
				fm.ActiveFilters = append(fm.ActiveFilters[:i], fm.ActiveFilters[i+1:]...)
				break
			}
		}
	} else {
		found := false
		for i, filter := range fm.ActiveFilters {
			if filter.Type == filterType {
				fm.ActiveFilters[i].Value = value
				found = true
				break
			}
		}

		if !found {
			fm.ActiveFilters = append(fm.ActiveFilters, Filter{
				Type:  filterType,
				Value: value,
			})
		}
	}

	if fm.OnFilterChanged != nil {
		fm.OnFilterChanged(fm.ActiveFilters)
	}
}

func (fm *FilterManager) ClearFilters() {
	fm.ActiveFilters = []Filter{}

	if fm.OnFilterChanged != nil {
		fm.OnFilterChanged(fm.ActiveFilters)
	}
}

func (fm *FilterManager) GetFilterValue(filterType string) (string, bool) {
	for _, filter := range fm.ActiveFilters {
		if filter.Type == filterType {
			return filter.Value, true
		}
	}
	return "", false
}

func (fm *FilterManager) HasActiveFilters() bool {
	return len(fm.ActiveFilters) > 0
}

func (fm *FilterManager) GetActiveFiltersText() string {
	if len(fm.ActiveFilters) == 0 {
		return "None"
	}

	parts := make([]string, 0, len(fm.ActiveFilters))
	for _, filter := range fm.ActiveFilters {
		parts = append(parts, fmt.Sprintf("%s=%s", filter.Type, filter.Value))
	}
	return strings.Join(parts, ", ")
}
