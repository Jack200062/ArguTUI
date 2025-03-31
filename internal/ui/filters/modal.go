package filters

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilterModalResult struct {
	Filters  []Filter
	Canceled bool
}

func ShowFilterModal(
	app *tview.Application,
	pages *tview.Pages,
	categories []FilterCategory,
	activeFilters []Filter,
	onResult func(FilterModalResult),
) {
	currentFilters := make([]Filter, len(activeFilters))
	copy(currentFilters, activeFilters)

	ui := createFilterUI(currentFilters)

	setupHandlers(app, pages, ui, categories, activeFilters, onResult)

	pages.AddPage("filter_modal", ui.flex, true, true)

	if len(categories) > 0 {
		ui.categoriesList.SetCurrentItem(0)
		ui.updateOptionsList(0)
	}

	app.SetFocus(ui.categoriesList)
}

type filterUI struct {
	flex           *tview.Flex
	modal          *tview.Flex
	modalFrame     *tview.Frame
	contentArea    *tview.Flex
	categoriesList *tview.List
	optionsList    *tview.List

	title             *tview.TextView
	activeFiltersView *tview.TextView
	footer            *tview.TextView

	updateActiveFiltersView func()
	updateOptionsList       func(int)

	currentFilters []Filter
}

func createFilterUI(currentFilters []Filter) *filterUI {
	ui := &filterUI{}

	ui.modal = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.currentFilters = currentFilters

	ui.title = tview.NewTextView()

	ui.activeFiltersView = tview.NewTextView().
		SetText("Active filters: None").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	ui.updateActiveFiltersView = func() {
		text := "Active filters: "
		if len(ui.currentFilters) == 0 {
			text += "[gray]None[-:-:-]"
		} else {
			filterTexts := make([]string, 0, len(ui.currentFilters))
			for _, f := range ui.currentFilters {
				filterTexts = append(filterTexts, f.Type+"="+f.Value)
			}
			text += "[#63a0bf]" + strings.Join(filterTexts, ", ") + "[-:-:-]"
		}
		ui.activeFiltersView.SetText(text)
	}
	ui.updateActiveFiltersView()

	ui.contentArea = tview.NewFlex().SetDirection(tview.FlexColumn)

	ui.categoriesList = tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.NewHexColor(0x373737)).
		SetSelectedTextColor(tcell.NewHexColor(0x00bebe))

	ui.optionsList = tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.NewHexColor(0x373737)).
		SetSelectedTextColor(tcell.NewHexColor(0x00bebe))
	ui.optionsList.SetBorder(true).
		SetBorderColor(tcell.NewHexColor(0x63a0bf))

	ui.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[#017be9]←/→[gray] Change Panel [#017be9]A/C [gray]Apply/Clear filters [#017be9]Enter[gray] Select  [#017be9]Esc[gray] Cancel").
		SetTextAlign(tview.AlignCenter)

	ui.contentArea.AddItem(ui.categoriesList, 0, 1, true)
	ui.contentArea.AddItem(ui.optionsList, 0, 2, false)

	ui.modal.AddItem(ui.title, 1, 0, false)
	ui.modal.AddItem(ui.activeFiltersView, 1, 0, false)
	ui.modal.AddItem(ui.contentArea, 0, 1, true)
	ui.modal.AddItem(ui.footer, 1, 0, false)

	ui.modalFrame = tview.NewFrame(ui.modal)
	ui.modalFrame.SetBorders(1, 1, 1, 1, 1, 1)
	ui.modalFrame.SetBorder(true)
	ui.modalFrame.SetTitle(" Filter Settings ")
	ui.modalFrame.SetTitleAlign(tview.AlignCenter)
	ui.modalFrame.SetTitleColor(tcell.ColorYellow)
	ui.modalFrame.SetBorderColor(tcell.NewHexColor(0x63a0bf))

	ui.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(ui.modalFrame, 60, 1, true).
				AddItem(nil, 0, 1, false),
			15, 1, true).
		AddItem(nil, 0, 1, false)

	return ui
}

func setupHandlers(
	app *tview.Application,
	pages *tview.Pages,
	ui *filterUI,
	categories []FilterCategory,
	activeFilters []Filter,
	onResult func(FilterModalResult),
) {
	addCategoryItems(ui, categories, onResult, pages)

	configureOptionsUpdater(ui, categories)

	setupCategorySelectionHandler(app, ui, categories)

	setupOptionsKeyHandler(app, ui)

	setupGlobalKeyHandler(ui, activeFilters, onResult, pages, ui.modalFrame)
}

func addCategoryItems(
	ui *filterUI,
	categories []FilterCategory,
	onResult func(FilterModalResult),
	pages *tview.Pages,
) {
	for _, category := range categories {
		ui.categoriesList.AddItem(category.Title, "", 0, nil)
	}
}

func configureOptionsUpdater(
	ui *filterUI,
	categories []FilterCategory,
) {
	ui.updateOptionsList = func(categoryIndex int) {
		ui.optionsList.Clear()
		category := categories[categoryIndex]

		activeValue := getFilterValue(ui.currentFilters, category.Type)
		addOptionsToList(ui, category, activeValue, ui.currentFilters, categoryIndex)

		ui.optionsList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			optionValue := mainText
			ui.currentFilters = applyFilter(ui.currentFilters, category.Type, optionValue)
			ui.updateActiveFiltersView()
		})
	}
}

func getFilterValue(filters []Filter, filterType string) string {
	for _, filter := range filters {
		if filter.Type == filterType {
			return filter.Value
		}
	}
	return ""
}

func addOptionsToList(
	ui *filterUI,
	category FilterCategory,
	activeValue string,
	currentFilters []Filter,
	categoryIndex int,
) {
	for _, option := range category.Options {
		isActive := option == activeValue
		displayText := option

		if isActive {
			displayText = "✓ " + displayText
		}

		ui.optionsList.AddItem(displayText, "", 0, nil)
	}
}

func applyFilter(filters []Filter, filterType, filterValue string) []Filter {
	for i, filter := range filters {
		if filter.Type == filterType {
			if filter.Value == filterValue {
				return append(filters[:i], filters[i+1:]...)
			}
			filters[i].Value = filterValue
			return filters
		}
	}
	return append(filters, Filter{
		Type:  filterType,
		Value: filterValue,
	})
}

func setupCategorySelectionHandler(
	app *tview.Application,
	ui *filterUI,
	categories []FilterCategory,
) {
	ui.categoriesList.SetChangedFunc(func(i int, _ string, _ string, _ rune) {
		if i < len(categories) {
			ui.updateOptionsList(i)
			ui.categoriesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEnter {
					app.SetFocus(ui.optionsList)
					return nil
				}
				return event
			})

			ui.categoriesList.SetSelectedFocusOnly(true)
		}
	})
}

func setupOptionsKeyHandler(app *tview.Application, ui *filterUI) {
	ui.optionsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft || event.Key() == tcell.KeyEscape {
			app.SetFocus(ui.categoriesList)
			return nil
		}
		return event
	})
}

func setupGlobalKeyHandler(
	ui *filterUI,
	activeFilters []Filter,
	onResult func(FilterModalResult),
	pages *tview.Pages,
	frame *tview.Frame,
) {
	ui.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			onResult(FilterModalResult{
				Filters:  activeFilters,
				Canceled: true,
			})
			pages.RemovePage("filter_modal")
			return nil
		}
		return event
	})

	ui.modalFrame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'a', 'A':
			onResult(FilterModalResult{
				Filters:  ui.currentFilters,
				Canceled: false,
			})
			pages.RemovePage("filter_modal")
			return nil
		case 'c', 'C':
			ui.currentFilters = []Filter{}
			ui.updateActiveFiltersView()
			return nil
		case 'f', 'F':
			ui.currentFilters = []Filter{}
			onResult(FilterModalResult{
				Filters:  ui.currentFilters,
				Canceled: false,
			})
			pages.RemovePage("filter_modal")
			return nil
		}
		return event
	})
}
