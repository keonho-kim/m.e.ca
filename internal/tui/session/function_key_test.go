// Purpose: Keep unassigned function keys from opening screens or changing the active task.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestUnassignedFunctionKey.
// Related: internal/tui/session/keys.go, internal/tui/session/render.go, internal/tui/view/footer.go.
package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestUnassignedFunctionKey(t *testing.T) {
	for _, width := range []int{32, 160} {
		for _, page := range []string{"chat", "model", "endpoint"} {
			t.Run(fmt.Sprintf("width=%d/page=%s", width, page), func(t *testing.T) {
				a := fixture(t)
				a.screen.SetSize(width, 24)
				if page == "model" {
					a.openModel()
				} else {
					a.page = page
				}
				a.editor.Set("keep draft")
				a.menuInput.Set("keep field")
				a.output.Append("existing conversation", false)
				a.busy, a.agentActive = true, true
				cancelled := false
				a.taskCancel = func() { cancelled = true }
				a.render()
				before := strings.Join(screenRows(a, 24), "\n")
				if strings.Contains(screenLine(a.screen, 23, width), "F1") {
					t.Fatal("unassigned key advertised in the footer")
				}
				a.keyEvent(tcell.NewEventKey(tcell.KeyF1, "", 0))
				a.render()
				if a.page != page || a.editor.String() != "keep draft" || a.menuInput.String() != "keep field" || cancelled || !a.busy {
					t.Fatal("unassigned key changed focus, input or running task")
				}
				if after := strings.Join(screenRows(a, 24), "\n"); after != before {
					t.Fatal("unassigned key opened or changed a screen")
				}
				if page == "chat" {
					a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
					if !cancelled {
						t.Fatal("unassigned key intercepted normal task cancellation")
					}
				}
			})
		}
	}
}
