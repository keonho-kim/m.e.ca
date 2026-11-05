// Purpose: Fit resident memory, tokens, optional cost and throughput beside the activity indicator.
// Pattern: Terminal renderer.
// Usage: Every screen uses Header; status-only frames reuse it without rebuilding body layout.
// Related: internal/tui/view/chat.go, internal/tui/view/menu.go, internal/tui/view/files.go.
package view

import (
	"fmt"
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/rivo/uniseg"
)

// Header replaces the top row, retaining unchanged cells and clearing only unused padding.
func Header(screen tcell.Screen, title string, usage modelapi.TokenUsage, width int, activity Activity) {
	tokens := fmt.Sprintf("%s %d · %s %d", activity.Language.Text("INPUT"), usage.Input, activity.Language.Text("OUTPUT"), usage.Output)
	rate := ""
	if activity.TokensPerSecond != nil {
		rate = fmt.Sprintf(" · %s %.1f", activity.Language.Text("TPS"), *activity.TokensPerSecond)
	}
	label := activity.Label()
	withMemory := tokens
	if activity.MemoryUsage != "" {
		withMemory = activity.MemoryUsage + " · " + tokens
	}
	if activity.Compressing && uniseg.StringWidth(label+withMemory)+1 > width {
		label = activity.indicatorLabel()
	}
	if activity.Compressing && uniseg.StringWidth(label+tokens)+1 > width {
		used := Text(screen, 0, 0, width, label, tcell.StyleDefault.Bold(true))
		screen.FillArea(used, 0, width-used, 1, ' ', tcell.StyleDefault)
		return
	}
	if uniseg.StringWidth(withMemory+label)+1 <= width {
		tokens = withMemory
	}
	if activity.ShowCost {
		cost := " · " + activity.Language.Cost(usage.Cost)
		if uniseg.StringWidth(tokens+cost+label)+1 <= width {
			tokens += cost
		}
	}
	if rate != "" && uniseg.StringWidth(tokens+rate+label)+1 <= width {
		tokens += rate
	}
	if title != "" && !activity.Compressing {
		label += " · " + title
	}
	x := width - uniseg.StringWidth(tokens)
	if x < 0 {
		used := Text(screen, 0, 0, width, label, tcell.StyleDefault)
		screen.FillArea(used, 0, width-used, 1, ' ', tcell.StyleDefault)
		return
	}
	used := Text(screen, 0, 0, max(0, x-1), label, tcell.StyleDefault.Bold(activity.Running).Dim(!activity.Running))
	screen.FillArea(used, 0, x-used, 1, ' ', tcell.StyleDefault)
	Text(screen, x, 0, width-x, tokens, tcell.StyleDefault.Dim(true))
}
