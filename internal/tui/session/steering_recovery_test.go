// Purpose: Preserve oversized undelivered steering without bypassing the editable-input budget.
// Pattern: Event-loop recovery regression.
// Usage: go test ./internal/tui/session -run TestOversizedSteering.
// Related: internal/tui/session/steering.go, internal/tui/composer/queue/recovery.go.
package session

import (
	"strings"
	"testing"

	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/input"
)

func TestOversizedSteeringRestoresSeparateRequests(t *testing.T) {
	a := screenFixture(t)
	a.compose.steering = make(chan requestinput.Request, 16)
	body := strings.Repeat("x", 9<<20)
	a.steerInput(body)
	a.steerInput(body)
	a.restoreSteering()
	if len(a.compose.editor.String()) > input.MaxBytes || len(a.compose.queued.Messages) != 2 {
		t.Fatal("steering recovery merged or lost accepted messages")
	}
	if !a.compose.queued.Paused {
		t.Fatal("oversized recovery did not pause automatic dispatch")
	}
	a.compose.editor.Set("current draft")
	a.finishPending(true)
	if len(a.compose.queued.Messages) != 2 || a.compose.editor.String() != "current draft" || a.work.state.active() {
		t.Fatal("another completion automatically dispatched recovered input")
	}
}
