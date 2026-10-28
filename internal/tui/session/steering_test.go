// Purpose: Route active-task input to steering without intercepting local file commands.
// Pattern: Terminal input contract tests.
// Usage: go test ./internal/tui/session -run TestSteeringKeepsLocalCommands.
// Related: internal/tui/session/commands.go, internal/tui/session/steering.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"testing"
)

func TestSteeringKeepsLocalCommands(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.steering = make(chan composer.Request, 1)
	a.editor.Set("Use the alternate approach")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	select {
	case text := <-a.steering:
		if text.Text != "Use the alternate approach" {
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
