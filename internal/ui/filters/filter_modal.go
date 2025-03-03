package filters

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ModalSize struct {
	Width  int
	Height int
}

type OverlayPosition struct {
	Top    int
	Left   int
	Right  int
	Bottom int
}

type BaseFilterModal struct {
	app            *tview.Application
	grid           *tview.Grid
	title          *tview.TextView
	content        tview.Primitive
	footer         *tview.TextView
	theme          ThemeColors
	onDone         FilterHandler
	returnApp      tview.Primitive
	size           ModalSize
	centered       bool
	currentFilters []FilterState
	asOverlay      bool
	overlayPages   *tview.Pages
	position       OverlayPosition
}

func NewBaseFilterModal(app *tview.Application, title string, returnApp tview.Primitive, theme ThemeColors) *BaseFilterModal {
	if theme.Background == tcell.ColorDefault {
		theme = DefaultTheme()
	}

	baseModal := &BaseFilterModal{
		app:       app,
		theme:     theme,
		returnApp: returnApp,
		centered:  true,
		asOverlay: false,
		size: ModalSize{
			Width:  50,
			Height: 15,
		},
		position: OverlayPosition{
			Top:    5,
			Left:   10,
			Right:  10,
			Bottom: 5,
		},
	}

	baseModal.grid = tview.NewGrid().
		SetRows(1, 0, 1).
		SetColumns(0).
		SetBorders(true)

	baseModal.title = tview.NewTextView().
		SetDynamicColors(true).
		SetText(StyleText(title, theme.HeaderText)).
		SetTextAlign(tview.AlignCenter)
	baseModal.title.SetBackgroundColor(theme.Background)

	baseModal.footer = CreateFooter(theme)

	baseModal.grid.SetBackgroundColor(theme.Background)
	baseModal.grid.SetBorderColor(theme.Border)

	return baseModal
}

func (m *BaseFilterModal) SetContent(content tview.Primitive) *BaseFilterModal {
	m.content = content
	return m
}

func (m *BaseFilterModal) SetDoneFunc(handler FilterHandler) *BaseFilterModal {
	m.onDone = handler
	return m
}

func (m *BaseFilterModal) SetSize(width, height int) *BaseFilterModal {
	m.size.Width = width
	m.size.Height = height
	return m
}

func (m *BaseFilterModal) SetCentered(centered bool) *BaseFilterModal {
	m.centered = centered
	return m
}

func (m *BaseFilterModal) SetCurrentFilters(filters []FilterState) *BaseFilterModal {
	m.currentFilters = filters
	return m
}

func (m *BaseFilterModal) SetAsOverlay(asOverlay bool) *BaseFilterModal {
	m.asOverlay = asOverlay
	return m
}

func (m *BaseFilterModal) SetPosition(top, left, right, bottom int) *BaseFilterModal {
	m.position.Top = top
	m.position.Left = left
	m.position.Right = right
	m.position.Bottom = bottom
	return m
}

func (m *BaseFilterModal) GetInputCapture(customHandler func(event *tcell.EventKey) *tcell.EventKey) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			if m.onDone != nil {
				result := FilterResult{
					Canceled: true,
					Filters:  m.currentFilters,
				}
				m.onDone(result)
			}
			m.Close()
			return nil
		}

		if customHandler != nil {
			return customHandler(event)
		}

		return event
	}
}

func (m *BaseFilterModal) Build() tview.Primitive {
	m.grid.AddItem(m.title, 0, 0, 1, 1, 0, 0, false)

	if m.content != nil {
		m.grid.AddItem(m.content, 1, 0, 1, 1, 0, 0, true)
	}

	m.grid.AddItem(m.footer, 2, 0, 1, 1, 0, 0, false)

	if m.centered {
		centeredLayout := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(
				tview.NewFlex().
					SetDirection(tview.FlexColumn).
					AddItem(nil, 0, 1, false).
					AddItem(m.grid, m.size.Width, 0, true).
					AddItem(nil, 0, 1, false),
				m.size.Height, 0, true).
			AddItem(nil, 0, 1, false)

		return centeredLayout
	}

	return m.grid
}

func (m *BaseFilterModal) Show() {
	if m.asOverlay {
		m.overlayPages = tview.NewPages().
			AddPage("background", m.returnApp, true, true).
			AddPage("modal", m.Build(), true, true)

		m.app.SetRoot(m.overlayPages, true)
	} else {
		m.app.SetRoot(m.Build(), true)
	}
}

func (m *BaseFilterModal) Close() {
	if m.asOverlay {
		m.app.SetRoot(m.returnApp, true)
	} else {
		m.app.SetRoot(m.returnApp, true)
	}
}
