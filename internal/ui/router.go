package ui

import "github.com/rivo/tview"

type Router struct {
	app     *tview.Application
	screens map[string]Screen
	current Screen
}

func NewRouter(app *tview.Application) *Router {
	return &Router{
		app:     app,
		screens: make(map[string]Screen),
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
		r.current = screen
		r.app.SetRoot(screen.Init(), true)
	}
}

func (r *Router) Current() Screen {
	return r.current
}
