package applicationlist

import (
	"fmt"

	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ScreenAppList реализует интерфейс ui.Screen – экран списка приложений.
type ScreenAppList struct {
	app          *tview.Application   // Ссылка на главный объект приложения.
	instanceInfo *common.InstanceInfo // Общая информация об инстансе.
	apps         []argocd.Application // Список приложений.
	table        *tview.Table         // Таблица для вывода данных.
	grid         *tview.Grid          // Корневой элемент экрана.
}

// New создаёт новый экран списка приложений, принимая ссылку на tview.Application.
func New(app *tview.Application, instanceInfo *common.InstanceInfo, apps []argocd.Application) *ScreenAppList {
	return &ScreenAppList{
		app:          app,
		instanceInfo: instanceInfo,
		apps:         apps,
	}
}

// Init инициализирует экран и возвращает его корневой элемент.
func (s *ScreenAppList) Init() tview.Primitive {
	// Верхняя панель с шорткатами.
	topBar := tview.NewTextView().
		SetText(" <TAB> Switch Panel   q Quit   d Details ").
		SetTextAlign(tview.AlignLeft)

	// Левая панель с информацией об ArgoCD инстансе.
	instanceBox := tview.NewTextView().
		SetText(s.instanceInfo.String())
	instanceBox.Box.SetBorder(true)
	instanceBox.Box.SetTitle(" ArgoCD Instance ")

	// Таблица для списка приложений.
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	table.Box.SetBorder(false)
	s.table = table

	// Заполняем заголовки столбцов.
	headers := []string{"Name", "Status", "Project"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[::b]%s", h)).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false)
		s.table.SetCell(0, col, cell)
	}

	// Заполняем данные.
	row := 1
	for _, app := range s.apps {
		s.table.SetCell(row, 0, tview.NewTableCell(app.Name))
		s.table.SetCell(row, 1, tview.NewTableCell(app.Status))
		s.table.SetCell(row, 2, tview.NewTableCell(app.Project))
		row++
	}

	// Компонуем элементы с помощью Grid.
	s.grid = tview.NewGrid().
		SetRows(1, 0).     // Верхняя строка для шорткатов, ниже – основная область.
		SetColumns(30, 0). // Левая колонка фиксированной ширины для instanceBox, правая – таблица.
		SetBorders(true)

	s.grid.AddItem(topBar, 0, 0, 1, 2, 0, 0, false).
		AddItem(instanceBox, 1, 0, 1, 1, 0, 0, false).
		AddItem(s.table, 1, 1, 1, 1, 0, 0, true)

	// Глобальная обработка клавиш.
	s.grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			// Вызываем Stop на главном приложении.
			s.app.Stop()
			return nil
		case 'd':
			// Здесь можно добавить переключение на экран деталей.
		}
		if event.Key() == tcell.KeyTAB {
			// Переключаем фокус между таблицей и instanceBox.
			if s.table.HasFocus() {
				s.app.SetFocus(instanceBox)
			} else {
				s.app.SetFocus(s.table)
			}
			return nil
		}
		return event
	})

	return s.grid
}

// Name возвращает имя экрана.
func (s *ScreenAppList) Name() string {
	return "ApplicationList"
}

// Проверка реализации интерфейса ui.Screen.
var _ ui.Screen = (*ScreenAppList)(nil)
