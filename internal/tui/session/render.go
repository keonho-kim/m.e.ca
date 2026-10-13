// Purpose: Coordinate session screens and bounded output projections from event-loop state.
// Pattern: Presentation renderer.
// Usage: Called by the terminal event loop after input or worker events.
// Related: internal/tui/session/app.go, internal/tui/viewport/viewport.go, internal/tui/session/render_files.go, internal/tui/session/render_hints.go, internal/tui/view/preview.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/view"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

func (a *app) render() {
	a.screen.Clear()
	a.screen.HideCursor()
	w, h := a.screen.Size()
	if w < 12 || h < 4 {
		view.Text(a.screen, 0, 0, w, "Resize terminal", tcell.StyleDefault)
		a.screen.Show()
		return
	}
	if a.page == "files" {
		a.renderFiles(w, h)
		a.screen.Show()
		return
	}
	if a.page != "chat" {
		a.renderMenu(w, h)
		a.screen.Show()
		return
	}
	view.Header(a.screen, a.selection.Model, a.selection.Thinking != nil && *a.selection.Thinking, a.settings, a.planMode, w)
	rows := view.Input(a.screen, a.editor, 1, w, min(a.opts.Config.UI.InputMaxLines, h-3))
	y := a.renderHints(1+rows, w, h)
	view.Text(a.screen, 0, y, w, strings.Repeat("─", w), tcell.StyleDefault)
	y++
	lines := a.output.Lines(w)
	appendTemporary := func(value string, thinking bool) {
		for _, line := range viewport.Wrap(value, w) {
			lines = append(lines, viewport.Line{Text: line, Thinking: thinking})
		}
	}
	if a.thinking != "" {
		appendTemporary(a.thinking, true)
	}
	if a.stream != "" {
		appendTemporary(a.stream, false)
	}
	if a.status != "" {
		appendTemporary(a.status, false)
	}
	start := a.output.Start(lines, h-y)
	for i := start; i < len(lines) && y < h; i++ {
		style := tcell.StyleDefault
		if lines[i].Thinking {
			style = style.Dim(true)
		}
		view.Text(a.screen, 0, y, w, lines[i].Text, style)
		y++
	}
	a.screen.Show()
}
func (a *app) renderMenu(w, h int) {
	title := a.page
	switch a.page {
	case "global":
		title = a.changeField + ": " + a.changeValue
	case "key":
		title = "API Key (OS credential store)"
	case "manual":
		title = "Model ID"
	case "planapproval":
		title = "Awaiting approval"
	case "approval":
		title = a.proposal.Name + " " + a.proposal.Target
	}
	view.Text(a.screen, 0, 0, w, title, tcell.StyleDefault)
	if a.page == "endpoint" || a.page == "key" || a.page == "manual" {
		view.Field(a.screen, a.menuInput, 2, w, a.page == "key")
		if a.status != "" {
			view.Text(a.screen, 0, 4, w, a.status, tcell.StyleDefault)
		}
		return
	}
	y := 2
	if a.page == "approval" && a.proposal != nil {
		lines := view.PreviewLines(a.proposal.Preview, w, a.proposal.Diff)
		a.previewOffset = min(a.previewOffset, max(0, len(lines)-(h-7)))
		for _, line := range lines[a.previewOffset:] {
			if y >= h-5 {
				break
			}
			style := tcell.StyleDefault
			if line.Change != 0 {
				style = view.ChangeStyle(line.Change)
				view.Text(a.screen, 0, y, w, strings.Repeat(" ", w), style)
			}
			view.Text(a.screen, 0, y, w, line.Text, style)
			y++
		}
	}
	if a.page == "planapproval" {
		for _, line := range viewport.Wrap(a.planText(), w) {
			if y >= h-5 {
				break
			}
			view.Text(a.screen, 0, y, w, line, tcell.StyleDefault)
			y++
		}
	}
	visible := max(1, h-y-2)
	start := max(0, a.choice-visible+1)
	for i := start; i < min(len(a.choices), start+visible); i++ {
		if a.page == "model" && i == 1 {
			view.Thinking(a.screen, y, w, a.thinkingChoice, a.selection.ThinkingSupported, i == a.choice)
		} else {
			view.Text(a.screen, 0, y, w, a.choices[i], view.SelectionStyle(i == a.choice))
		}
		y++
	}
	if a.status != "" && y < h {
		view.Text(a.screen, 0, y, w, a.status, tcell.StyleDefault)
	}
}
