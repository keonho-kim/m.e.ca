// Purpose: Fit token counts beside run activity while keeping compaction visible in narrow headers.
// Pattern: Terminal renderer.
// Usage: Chat, menu and file views share topLine while retaining their own content layout.
// Related: internal/tui/view/chat.go, internal/tui/view/menu.go, internal/tui/view/files.go.
package view

import (
	"fmt"
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/rivo/uniseg"
)

func topLine(screen tcell.Screen, title string, usage modelapi.TokenUsage, width int, activity Activity) {
	tokens := fmt.Sprintf("%s %d · %s %d", activity.Language.Text("INPUT"), usage.Input, activity.Language.Text("OUTPUT"), usage.Output)
	rate := " · " + activity.Language.Text("TPS") + " —"
	if activity.TokensPerSecond != nil {
		rate = fmt.Sprintf(" · %s %.1f", activity.Language.Text("TPS"), *activity.TokensPerSecond)
	}
	label := activity.Label()
	if activity.Compressing && uniseg.StringWidth(label+tokens)+1 > width {
		label = activity.compressionLabel()
	}
	if activity.Compressing && uniseg.StringWidth(label+tokens)+1 > width {
		Text(screen, 0, 0, width, label, tcell.StyleDefault.Bold(true))
		return
	}
	if uniseg.StringWidth(tokens+rate+label)+1 <= width {
		tokens += rate
	}
	if title != "" && !activity.Compressing {
		label += " · " + title
	}
	x := width - uniseg.StringWidth(tokens)
	if x < 0 {
		Text(screen, 0, 0, width, label, tcell.StyleDefault)
		return
	}
	Text(screen, 0, 0, max(0, x-1), label, tcell.StyleDefault.Bold(activity.Running).Dim(!activity.Running))
	Text(screen, x, 0, width-x, tokens, tcell.StyleDefault.Dim(true))
}
