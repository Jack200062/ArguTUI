package components

import (
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
)

// EditorModal creates a simple full-screen editor with Save/Cancel.
// Controls: Ctrl-S — save, Esc — cancel.
func EditorModal(title, initial string, onSave func(text string), onCancel func()) tview.Primitive {
    area := tview.NewTextArea().
        SetText(initial, true).
        SetPlaceholder("Edit YAML…")
    area.SetBorder(true).SetTitle(title)

    help := tview.NewTextView().
        SetDynamicColors(true).
        SetText("[darkgray]Ctrl-S: Save  |  Esc: Cancel[-]")
    help.SetBorder(true)

    layout := tview.NewGrid().
        SetRows(-1, 1).
        SetColumns(-1)
    layout.AddItem(area, 0, 0, 1, 1, 0, 0, true)
    layout.AddItem(help, 1, 0, 1, 1, 0, 0, false)

    layout.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
        if ev.Key() == tcell.KeyEsc {
            if onCancel != nil {
                onCancel()
            }
            return nil
        }
        if ev.Key() == tcell.KeyCtrlS {
            if onSave != nil {
                onSave(area.GetText())
            }
            return nil
        }
        return ev
    })

    return layout
}


