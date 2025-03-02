package ui

import (
	"github.com/rivo/tview"
)

type Router struct {
	app     *tview.Application
	screens map[string]Screen
	current Screen
	history []string
}

func NewRouter(app *tview.Application) *Router {
	return &Router{
		app:     app,
		screens: make(map[string]Screen),
		history: make([]string, 0),
	}
}

func (r *Router) AddScreen(s Screen) {
	r.screens[s.Name()] = s
	if r.current == nil {
		r.current = s
	}
}

func (r *Router) SwitchTo(name string) {
	if screen, ok := r.screens[name]; ok {
		if r.current != nil {
			r.history = append(r.history, r.current.Name())
		}
		r.current = screen
		r.app.SetRoot(screen.Init(), true)
	}
}

func (r *Router) Back() {
	if len(r.history) == 0 {
		return
	}
	prevName := r.history[len(r.history)-1]
	r.history = r.history[:len(r.history)-1]

	if screen, ok := r.screens[prevName]; ok {
		r.current = screen
		r.app.SetRoot(screen.Init(), true)
	}
}

func (r *Router) Current() Screen {
	return r.current
}
