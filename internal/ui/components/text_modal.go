package components

import (
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
)

// TextModal is a simple scrollable modal for showing large text (YAML/diff)
func TextModal(title string, content string, onClose func()) tview.Primitive {
    text := tview.NewTextView().
        SetDynamicColors(false).
        SetScrollable(true).
        SetWrap(false).
        SetText(content)
    text.SetBorder(true).SetTitle(title)
    text.SetBackgroundColor(tcell.NewHexColor(0x000000))
    text.SetTitleColor(tcell.ColorYellow)
    text.SetBorderColor(tcell.NewHexColor(0x63a0bf))

    // capture Esc / q to close
    text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
            if onClose != nil {
                onClose()
            }
            return nil
        }
        return event
    })
    return text
}

// TextModalColored is like TextModal but enables tview color tags
func TextModalColored(title string, content string, onClose func()) tview.Primitive {
    text := tview.NewTextView().
        SetDynamicColors(true).
        SetScrollable(true).
        SetWrap(false).
        SetText(content)
    text.SetBorder(true).SetTitle(title)
    text.SetBackgroundColor(tcell.NewHexColor(0x000000))
    text.SetTitleColor(tcell.ColorYellow)
    text.SetBorderColor(tcell.NewHexColor(0x63a0bf))

    text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
            if onClose != nil {
                onClose()
            }
            return nil
        }
        return event
    })
    return text
}


