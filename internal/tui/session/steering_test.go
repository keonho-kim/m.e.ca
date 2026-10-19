// Purpose: Route active-task input to steering without intercepting local file commands.
// Pattern: Terminal input contract tests.
// Usage: go test ./internal/tui/session -run TestSteeringKeepsLocalCommands.
// Related: internal/tui/session/commands.go, internal/tui/session/steering.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"testing"
)

func TestSteeringKeepsLocalCommands(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive = true, true
	a.steering = make(chan string, 1)
	a.editor.Set("Use the alternate approach")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	select {
	case text := <-a.steering:
		if text != "Use the alternate approach" {
			t.Fatal(text)
		}
	default:
		t.Fatal("message was not queued")
	}
	if a.editor.String() != "" || len(a.steeringDrafts) != 1 {
		t.Fatal("queued draft was lost")
	}
	a.editor.Set("!dir")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "files" && !a.directoryLoading {
		t.Fatal("local command was swallowed by steering")
	}
}
