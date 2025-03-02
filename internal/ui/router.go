package ui

import (
	"errors"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Screen представляет экран приложения
type Screen interface {
	Name() string
	Init() tview.Primitive
}

// Router управляет навигацией между экранами
type Router struct {
	app     *tview.Application
	screens map[string]Screen
	current Screen
	history []string
	mutex   sync.RWMutex // защита от гонок при параллельном доступе
	isModal bool         // флаг, показывающий активно ли модальное окно
}

// NewRouter создает новый экземпляр роутера
func NewRouter(app *tview.Application) *Router {
	if app == nil {
		panic("tview.Application cannot be nil")
	}
	return &Router{
		app:     app,
		screens: make(map[string]Screen),
		history: make([]string, 0),
	}
}

// AddScreen добавляет экран в роутер
func (r *Router) AddScreen(s Screen) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if s == nil {
		return errors.New("screen cannot be nil")
	}

	name := s.Name()
	if name == "" {
		return errors.New("screen name cannot be empty")
	}

	if _, exists := r.screens[name]; exists {
		return errors.New("screen with this name already exists")
	}

	r.screens[name] = s
	if r.current == nil {
		r.current = s
	}
	return nil
}

// SwitchTo переключает на указанный экран
func (r *Router) SwitchTo(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isModal {
		return errors.New("cannot switch screens while modal is active")
	}

	screen, ok := r.screens[name]
	if !ok {
		return errors.New("screen not found: " + name)
	}

	if r.current != nil {
		r.history = append(r.history, r.current.Name())
	}
	r.current = screen
	r.app.SetRoot(screen.Init(), true)
	return nil
}

// Back возвращает к предыдущему экрану
func (r *Router) Back() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isModal {
		return errors.New("cannot go back while modal is active")
	}

	if len(r.history) == 0 {
		return errors.New("history is empty")
	}

	prevName := r.history[len(r.history)-1]
	r.history = r.history[:len(r.history)-1]

	screen, ok := r.screens[prevName]
	if !ok {
		return errors.New("previous screen not found: " + prevName)
	}

	r.current = screen
	r.app.SetRoot(screen.Init(), true)
	return nil
}

// Current возвращает текущий экран
func (r *Router) Current() Screen {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.current
}

// ShowModal отображает модальное окно и запоминает текущий экран
func (r *Router) ShowModal(modal tview.Primitive) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.isModal = true
	r.app.SetRoot(modal, false)
}

// CloseModal закрывает модальное окно и возвращается к предыдущему экрану
func (r *Router) CloseModal() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if !r.isModal || r.current == nil {
		return
	}
	r.isModal = false
	r.app.SetRoot(r.current.Init(), true)
}

// IsModalActive возвращает true, если активно модальное окно
func (r *Router) IsModalActive() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isModal
}

// ResetHistory очищает историю навигации
func (r *Router) ResetHistory() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.history = make([]string, 0)
}

// SetGlobalInputCapture устанавливает глобальный обработчик ввода
func (r *Router) SetGlobalInputCapture(callback func(event *tcell.EventKey) *tcell.EventKey) {
	r.app.SetInputCapture(callback)
}
