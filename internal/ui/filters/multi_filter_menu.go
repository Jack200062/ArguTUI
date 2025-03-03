package filters

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilterCategory struct {
	Title   string
	Type    FilterType
	Options []string
	// Map для быстрого поиска шорткатов
	Shortcuts map[string]rune
}

type MultiFilterOptions struct {
	Title         string
	ActiveFilters []FilterState
	Categories    []FilterCategory
	AsOverlay     bool
	Position      OverlayPosition
}

func GetActiveFiltersText(filters []FilterState) string {
	if len(filters) == 0 {
		return "None"
	}

	var parts []string
	for _, filter := range filters {
		parts = append(parts, fmt.Sprintf("%s=%s", filter.Type, filter.Value))
	}
	return strings.Join(parts, ", ")
}

func ShowMultiFilterMenu(
	app *tview.Application,
	options MultiFilterOptions,
	returnApp tview.Primitive,
	onDone FilterHandler,
) {
	theme := DefaultTheme()

	currentFilters := make([]FilterState, len(options.ActiveFilters))
	copy(currentFilters, options.ActiveFilters)

	modal := NewBaseFilterModal(app, options.Title, returnApp, theme).
		SetCurrentFilters(currentFilters).
		SetSize(60, 20)

	if options.AsOverlay {
		modal.SetAsOverlay(true)
		if options.Position.Top > 0 || options.Position.Left > 0 ||
			options.Position.Right > 0 || options.Position.Bottom > -0 {
			modal.SetPosition(
				options.Position.Top,
				options.Position.Left,
				options.Position.Right,
				options.Position.Bottom,
			)
		}
	}

	content := tview.NewFlex().SetDirection(tview.FlexRow)

	filtersText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("Active filters: " + StyleText(GetActiveFiltersText(currentFilters), theme.HeaderText)).
		SetTextAlign(tview.AlignCenter)
	filtersText.SetBackgroundColor(theme.Background)
	content.AddItem(filtersText, 1, 0, false)

	categoryList := tview.NewList().
		SetSelectedBackgroundColor(theme.Selection).
		SetSelectedTextColor(theme.SelectionText).
		SetMainTextColor(theme.Text).
		SetHighlightFullLine(true)
	categoryList.SetBorder(false)
	categoryList.SetBackgroundColor(theme.Background)

	updateFiltersText := func() {
		filtersText.SetText("Active filters: " + StyleText(GetActiveFiltersText(currentFilters), theme.HeaderText))
	}

	rightPanel := tview.NewFlex()
	rightPanel.SetBackgroundColor(theme.Background)

	categoryViews := make(map[FilterType]*tview.List)

	for _, category := range options.Categories {
		localCategory := category

		optionsList := tview.NewList().
			SetSelectedBackgroundColor(theme.Selection).
			SetSelectedTextColor(theme.SelectionText).
			SetMainTextColor(theme.Text).
			SetHighlightFullLine(true)
		optionsList.SetBorder(true).
			SetTitle(" " + category.Title + " ").
			SetTitleAlign(tview.AlignCenter).
			SetTitleColor(theme.HeaderText)
		optionsList.SetBackgroundColor(theme.Background)

		optionsList.AddItem("All (clear filter)", "", 'a', func() {
			currentFilters = UpdateFilter(currentFilters, localCategory.Type, "")
			updateFiltersText()
		})

		currentValue, _ := FindFilterByType(currentFilters, localCategory.Type)

		for _, option := range localCategory.Options {
			localOption := option

			var shortcut rune = 0
			if shortcutRune, ok := localCategory.Shortcuts[option]; ok {
				shortcut = shortcutRune
			}

			displayText := option

			if localCategory.Type == HealthFilter {
				displayText = StyleText(option, GetHealthStatusColor(option))
			} else if localCategory.Type == SyncFilter {
				displayText = StyleText(option, GetSyncStatusColor(option))
			}

			if currentValue == option {
				displayText = "✓ " + displayText
			}

			optionsList.AddItem(displayText, "", shortcut, func() {
				currentFilters = UpdateFilter(currentFilters, localCategory.Type, localOption)

				for i, opt := range localCategory.Options {
					text := opt

					if localCategory.Type == HealthFilter {
						text = StyleText(opt, GetHealthStatusColor(opt))
					} else if localCategory.Type == SyncFilter {
						text = StyleText(opt, GetSyncStatusColor(opt))
					}

					if opt == localOption {
						text = "✓ " + text
					}

					optionsList.SetItemText(i+1, text, "")
				}

				updateFiltersText()
			})
		}

		optionsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEnter {
				if onDone != nil {
					onDone(FilterResult{
						Canceled: false,
						Filters:  currentFilters,
					})
				}
				modal.Close()
				return nil
			}

			if event.Key() == tcell.KeyRune && (event.Rune() == 'b' || event.Rune() == 'B') {
				app.SetFocus(categoryList)
				return nil
			}

			if event.Key() == tcell.KeyLeft {
				app.SetFocus(categoryList)
				return nil
			}

			if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyEscape {
				app.SetFocus(categoryList)
				return nil
			}

			if event.Key() == tcell.KeyRune {
				if event.Rune() == 'a' || event.Rune() == 'A' {
					currentFilters = UpdateFilter(currentFilters, localCategory.Type, "")

					for i, opt := range localCategory.Options {
						text := opt

						if localCategory.Type == HealthFilter {
							text = StyleText(opt, GetHealthStatusColor(opt))
						} else if localCategory.Type == SyncFilter {
							text = StyleText(opt, GetSyncStatusColor(opt))
						}

						optionsList.SetItemText(i+1, text, "")
					}

					updateFiltersText()
					return nil
				}

				for i, opt := range localCategory.Options {
					if shortcutRune, ok := localCategory.Shortcuts[opt]; ok {
						if event.Rune() == shortcutRune {
							optionsList.SetCurrentItem(i + 1) // +1 для учета "All"

							currentFilters = UpdateFilter(currentFilters, localCategory.Type, opt)

							for j, option := range localCategory.Options {
								text := option

								if localCategory.Type == HealthFilter {
									text = StyleText(option, GetHealthStatusColor(option))
								} else if localCategory.Type == SyncFilter {
									text = StyleText(option, GetSyncStatusColor(option))
								}

								if option == opt {
									text = "✓ " + text
								}

								optionsList.SetItemText(j+1, text, "")
							}

							updateFiltersText()
							return nil
						}
					}
				}
			}

			return event
		})

		categoryViews[localCategory.Type] = optionsList
	}

	showCategoryOptions := func(categoryType FilterType) {
		if list, ok := categoryViews[categoryType]; ok {
			rightPanel.Clear()
			rightPanel.AddItem(list, 0, 1, true)
			app.SetFocus(list)
		}
	}

	for _, category := range options.Categories {
		localCategory := category
		categoryList.AddItem(category.Title, "", 0, func() {
			showCategoryOptions(localCategory.Type)
		})
	}

	categoryList.AddItem("Apply Filters", "", 0, func() {
		if onDone != nil {
			onDone(FilterResult{
				Canceled: false,
				Filters:  currentFilters,
			})
		}
		modal.Close()
	})

	categoryList.AddItem("Clear All Filters", "", 0, func() {
		currentFilters = []FilterState{}
		updateFiltersText()
	})

	categoryList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			if categoryList.GetCurrentItem() == len(options.Categories) {
				if onDone != nil {
					onDone(FilterResult{
						Canceled: false,
						Filters:  currentFilters,
					})
				}
				modal.Close()
				return nil
			}
			if categoryList.GetCurrentItem() == len(options.Categories)+1 {
				currentFilters = []FilterState{}
				updateFiltersText()
				return nil
			}

			if categoryList.GetCurrentItem() < len(options.Categories) {
				showCategoryOptions(options.Categories[categoryList.GetCurrentItem()].Type)
				return nil
			}
		}

		if event.Rune() == 'f' {
			modal.Close()
			return nil
		}

		if event.Key() == tcell.KeyRight && categoryList.GetCurrentItem() < len(options.Categories) {
			showCategoryOptions(options.Categories[categoryList.GetCurrentItem()].Type)
			return nil
		}

		return event
	})

	leftPane := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(categoryList, 0, 1, true)

	if len(options.Categories) == 0 {
		noCategories := tview.NewTextView().
			SetText("No filter categories available").
			SetTextAlign(tview.AlignCenter)
		noCategories.SetBackgroundColor(theme.Background)
		rightPanel.AddItem(noCategories, 0, 1, false)
	} else {
		rightPanel.AddItem(categoryViews[options.Categories[0].Type], 0, 1, true)
	}

	splitView := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftPane, 15, 1, true).
		AddItem(rightPanel, 0, 2, false)

	content.AddItem(filtersText, 1, 0, false)
	content.AddItem(splitView, 0, 1, true)

	modal.footer = CreateFooterWithNavigation(theme)

	modal.SetContent(content).
		SetDoneFunc(func(result FilterResult) {
			if onDone != nil {
				onDone(result)
			}
		}).
		Show()
}

func CreateFooterWithNavigation(theme ThemeColors) *tview.TextView {
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetText(StyleText("↑/↓", theme.ShortcutKey) +
			StyleText(" Navigate  ", tcell.ColorGray) +
			StyleText("←/→", theme.ShortcutKey) +
			StyleText(" Change panel  ", tcell.ColorGray) +
			StyleText("b", theme.ShortcutKey) +
			StyleText(" Back  ", tcell.ColorGray) +
			StyleText("Enter", theme.ShortcutKey) +
			StyleText(" Apply  ", tcell.ColorGray)).
		SetTextAlign(tview.AlignCenter)
	footer.SetBackgroundColor(theme.Background)
	return footer
}
