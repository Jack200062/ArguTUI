package components

import (
    "strings"

    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
)

// CommandHandler executes a parsed command string
type CommandHandler func(cmd string, args []string)

// CommandBar implements a simple k9s-like ':' command mode
type CommandBar struct {
    input   *tview.InputField
    onExec  CommandHandler
    history []string
}

func NewCommandBar(onExec CommandHandler) *CommandBar {
    inp := tview.NewInputField().
        SetLabel(": ").
        SetFieldTextColor(tcell.ColorWhite).
        SetFieldBackgroundColor(tcell.ColorBlack)
    c := &CommandBar{input: inp, onExec: onExec, history: []string{}}
    inp.SetDoneFunc(func(key tcell.Key) {
        if key == tcell.KeyEnter {
            text := strings.TrimSpace(inp.GetText())
            if text != "" && c.onExec != nil {
                parts := strings.Fields(text)
                c.onExec(parts[0], parts[1:])
                c.history = append(c.history, text)
            }
            // clear and hide handled by parent
        }
    })
    return c
}

func (c *CommandBar) Primitive() *tview.InputField { return c.input }


