package components

import (
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NewOctopusLogo renders a logo. If LOGO_PATH is set, it will load ASCII from file to match a given mock.
func NewOctopusLogo() *tview.TextView {
	if p := os.Getenv("LOGO_PATH"); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			tv := tview.NewTextView().
				SetDynamicColors(true).
				SetText(string(b)).
				SetTextAlign(tview.AlignCenter)
			tv.SetBackgroundColor(tcell.NewHexColor(0x000000))
			return tv
		}
	}
	body := "#f5e6c8"
	acc := "#017be9"
	art := strings.Join([]string{
		"      [" + body + "]   ____   ____   [-]",
		"      [" + body + "]  / __ \\ / __ \\  [-]",
		"      [" + body + "] | |  ) (  o  ) | [-]",
		"      [" + body + "] | |   \\(+)/   | [-]",
		"      [" + body + "] | |          | [-]",
		"      [" + body + "] | |__  ____  | [-]",
		"      [" + body + "]  \\___)(____)(_/ [-]",
		"      [" + acc + "] ////[" + body + "]  \\  [" + acc + "]////  __[-]",
		"      [" + acc + "]//// [" + body + "] xx [" + acc + "] \\//// __[-]",
		"",
		"[white::b]ArgoTUI[-]",
	}, "\n")

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetText(art).
		SetTextAlign(tview.AlignCenter)
	tv.SetBackgroundColor(tcell.NewHexColor(0x000000))
	return tv
}

// NewMiniLogoRight returns a single‑line compact logo for top bars
func NewMiniLogoRight() *tview.TextView {
	txt := "[#f5e6c8]〇[-][#017be9]八八[-] ArgoTUI"
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetText(txt).
		SetTextAlign(tview.AlignRight)
	tv.SetBackgroundColor(tcell.NewHexColor(0x000000))
	return tv
}
