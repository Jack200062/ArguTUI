package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	dmpkg "github.com/sergi/go-diff/diffmatchpatch"
)

// NewSideBySideDiff renders a split view diff similar to ArgoCD: Desired (left) vs Live (right).
// Expects already normalized text (e.g., YAML). Supports tview color tags in content.
func NewSideBySideDiff(title, leftTitle, rightTitle, leftText, rightText string, onClose func()) tview.Primitive {
	dmp := dmpkg.New()
	a, b, lineArray := dmp.DiffLinesToChars(leftText, rightText)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	type row struct {
		left         string
		right        string
		leftChanged  bool
		rightChanged bool
	}

	var rows []row
	for i := 0; i < len(diffs); i++ {
		d := diffs[i]
		switch d.Type {
		case dmpkg.DiffEqual:
			lines := strings.Split(strings.TrimSuffix(d.Text, "\n"), "\n")
			for _, l := range lines {
				rows = append(rows, row{left: l, right: l})
			}
		case dmpkg.DiffDelete:
			delLines := strings.Split(strings.TrimSuffix(d.Text, "\n"), "\n")
			var insLines []string
			if i+1 < len(diffs) && diffs[i+1].Type == dmpkg.DiffInsert {
				insLines = strings.Split(strings.TrimSuffix(diffs[i+1].Text, "\n"), "\n")
				i++ // consume insert as a replace
			}
			max := len(delLines)
			if len(insLines) > max {
				max = len(insLines)
			}
			for k := 0; k < max; k++ {
				var lStr, rStr string
				var lCh, rCh bool
				if k < len(delLines) {
					lStr = delLines[k]
					lCh = true
				}
				if k < len(insLines) {
					rStr = insLines[k]
					rCh = true
				}
				rows = append(rows, row{left: lStr, right: rStr, leftChanged: lCh, rightChanged: rCh})
			}
		case dmpkg.DiffInsert:
			insLines := strings.Split(strings.TrimSuffix(d.Text, "\n"), "\n")
			for _, l := range insLines {
				rows = append(rows, row{right: l, rightChanged: true})
			}
		}
	}

	leftBuilder := &strings.Builder{}
	rightBuilder := &strings.Builder{}
	leftLineNum := 1
	rightLineNum := 1
	for _, r := range rows {
		if r.left != "" || r.leftChanged {
			if r.leftChanged {
				fmt.Fprintf(leftBuilder, "[#ff6b6b]%4d  %s[-]\n", leftLineNum, r.left)
			} else {
				fmt.Fprintf(leftBuilder, "%4d  %s\n", leftLineNum, r.left)
			}
			leftLineNum++
		} else {
			fmt.Fprintln(leftBuilder)
		}

		if r.right != "" || r.rightChanged {
			if r.rightChanged {
				fmt.Fprintf(rightBuilder, "[#51cf66]%4d  %s[-]\n", rightLineNum, r.right)
			} else {
				fmt.Fprintf(rightBuilder, "%4d  %s\n", rightLineNum, r.right)
			}
			rightLineNum++
		} else {
			fmt.Fprintln(rightBuilder)
		}
	}

	leftView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(true)
	leftView.SetBorder(true).
		SetBorderColor(tcell.NewHexColor(0x63a0bf)).
		SetTitleAlign(tview.AlignCenter)
	leftView.SetTitle(fmt.Sprintf("[#ff6b6b]%s[-]", leftTitle))
	leftView.SetText(leftBuilder.String())

	rightView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(true)
	rightView.SetBorder(true).
		SetBorderColor(tcell.NewHexColor(0x63a0bf)).
		SetTitleAlign(tview.AlignCenter)
	rightView.SetTitle(fmt.Sprintf("[#51cf66]%s[-]", rightTitle))
	rightView.SetText(rightBuilder.String())

	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	header.SetBackgroundColor(tcell.NewHexColor(0x000000))
	// Project-like header styling with accent and key hint
	header.SetText(fmt.Sprintf("[#63a0bf::b]»[-] [#00bebe::b]%s[-]    [#017be9::b]esc[-] [darkgray]to close[-]", title))

	grid := tview.NewGrid().
		SetRows(1, -1).
		SetColumns(-1, -1)
	grid.AddItem(header, 0, 0, 1, 2, 0, 0, false)
	grid.AddItem(leftView, 1, 0, 1, 1, 0, 0, true)
	grid.AddItem(rightView, 1, 1, 1, 1, 0, 0, false)

	// sync vertical scroll
	syncScroll := func(delta int) {
		lx, ly := leftView.GetScrollOffset()
		rx, ry := rightView.GetScrollOffset()
		leftView.ScrollTo(lx, ly+delta)
		rightView.ScrollTo(rx, ry+delta)
	}
	grid.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEsc:
			if onClose != nil {
				onClose()
			}
			return nil
		case tcell.KeyUp:
			syncScroll(-1)
			return nil
		case tcell.KeyDown:
			syncScroll(1)
			return nil
		case tcell.KeyPgUp:
			syncScroll(-10)
			return nil
		case tcell.KeyPgDn:
			syncScroll(10)
			return nil
		case tcell.KeyHome:
			leftView.ScrollToBeginning()
			rightView.ScrollToBeginning()
			return nil
		case tcell.KeyEnd:
			leftView.ScrollToEnd()
			rightView.ScrollToEnd()
			return nil
		}
		return ev
	})

	return grid
}
