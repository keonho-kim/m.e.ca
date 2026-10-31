// Purpose: Keep approval input authoritative while preserving a suspended file browser.
// Pattern: Virtual terminal state-transition tests.
// Usage: go test ./internal/tui/session -run TestApprovalSuspendsFileViewer.
// Related: internal/tui/session/events.go, internal/tui/session/keys.go, internal/tui/session/menu_keys.go.
package session

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/preview"
)

func TestApprovalSuspendsFileViewer(t *testing.T) {
	for _, key := range []tcell.Key{tcell.KeyEscape, tcell.KeyEnter} {
		name := "reject"
		if key == tcell.KeyEnter {
			name = "approve"
		}
		t.Run(name, func(t *testing.T) {
			a := screenFixture(t)
			a.page, a.worker = "files", workerAgent
			a.filePreview = &preview.State{Path: "README.md", Content: "# Preview", Full: true}
			a.editor.Set("unsent draft")
			a.fileSubmission = &fileSubmission{}
			a.decision = make(chan bool, 1)
			a.receive(notice{Approval: &toolapi.Approval{Name: "protected operation"}})
			if keys := keymapLabels(t, a); keys["Esc"] != "거부" || keys["Shift+R"] != "" {
				t.Fatalf("approval inherited viewer shortcuts: %v", keys)
			}
			a.keyEvent(tcell.NewEventKey(key, "", 0))
			select {
			case approved := <-a.decision:
				if approved != (key == tcell.KeyEnter) {
					t.Fatal("approval input was handled by the suspended viewer")
				}
			default:
				t.Fatal("approval did not receive its decision")
			}
			if a.page != "files" || a.filePreview == nil || !a.filePreview.Full || a.editor.String() != "unsent draft" {
				t.Fatal("approval lost the suspended viewer or draft")
			}
		})
	}
}
