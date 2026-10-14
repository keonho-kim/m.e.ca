// Purpose: Render a fixed top row with a right-aligned uppercase numeric token counter.
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

func topLine(screen tcell.Screen, title string, usage modelapi.TokenUsage, width int) {
	label := fmt.Sprintf("INPUT %d OUTPUT %d", usage.Input, usage.Output)
	x := width - uniseg.StringWidth(label)
	if x < 0 {
		Text(screen, 0, 0, width, title, tcell.StyleDefault)
		return
	}
	Text(screen, 0, 0, max(0, x-1), title, tcell.StyleDefault)
	Text(screen, x, 0, width-x, label, tcell.StyleDefault.Dim(true))
}
