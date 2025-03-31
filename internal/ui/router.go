package ui

import (
	"errors"
	"sync"
	"time"

	"github.com/rivo/tview"
)

const ScreenExpirationTime = time.Hour

type Screen interface {
	Name() string
	Init() tview.Primitive
}

type Router struct {
	app            *tview.Application
	screens        map[string]Screen
	current        Screen
	history        []string
	mutex          sync.RWMutex
	isModal        bool
	screenCreation map[string]time.Time
	cleanupTicker  *time.Ticker
}

func NewRouter(app *tview.Application) *Router {
	if app == nil {
		panic("tview.Application cannot be nil")
	}

	r := &Router{
		app:            app,
		screens:        make(map[string]Screen),
		history:        make([]string, 0),
		screenCreation: make(map[string]time.Time),
	}

	r.cleanupTicker = time.NewTicker(10 * time.Minute)
	go r.cleanupRoutine()

	return r
}

func (r *Router) cleanupRoutine() {
	for range r.cleanupTicker.C {
		r.cleanupExpiredScreens()
	}
}

func (r *Router) Shutdown() {
	if r.cleanupTicker != nil {
		r.cleanupTicker.Stop()
	}
}

func (r *Router) cleanupExpiredScreens() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	for name, creationTime := range r.screenCreation {
		if now.Sub(creationTime) > ScreenExpirationTime {
			if r.current != nil && r.current.Name() == name {
				r.screenCreation[name] = now
				continue
			}

			delete(r.screens, name)
			delete(r.screenCreation, name)

			for i := 0; i < len(r.history); i++ {
				if r.history[i] == name {
					r.history = append(r.history[:i], r.history[i+1:]...)
					i--
				}
			}
		}
	}
}

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
	r.screenCreation[name] = time.Now()

	if r.current == nil {
		r.current = s
	}
	return nil
}

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
	r.screenCreation[name] = time.Now()
	r.app.SetRoot(screen.Init(), true)
	return nil
}

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

func (r *Router) IsModalActive() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isModal
}

func (r *Router) Current() Screen {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.current
}

func (r *Router) ShowModal(modal tview.Primitive) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.isModal = true
	r.app.SetRoot(modal, false)
}

func (r *Router) CloseModal() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if !r.isModal || r.current == nil {
		return
	}
	r.isModal = false
	r.app.SetRoot(r.current.Init(), true)
}
