// Purpose: Keep native selection and event attribution independent of compatible execution sessions.
// Pattern: Cross-adapter ownership regression test.
// Usage: go test ./internal/serverapi -run TestCompatibleRequestKeepsNativeSelection.
// Related: internal/commands/server/submission_session.go, internal/commands/server/events.go.
package serverapi

import (
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
)

func TestCompatibleRequestKeepsNativeSelection(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	native, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := f.app.Submit(t.Context(), server.Submission{Kind: "response", Stored: true, Key: "isolated", Messages: []server.InputMessage{{Role: "user", Text: "hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.WaitResponse(t.Context(), accepted.Response.ID); err != nil {
		t.Fatal(err)
	}
	state := f.app.State()
	if state.Session.ID != native.Session.ID || accepted.Response.Session == native.Session.ID {
		t.Fatal("compatible execution changed native selection", state.Session.ID, accepted.Response.Session)
	}
	batch, err := f.app.Replay(native.Cursor)
	if err != nil || batch.Snapshot != nil {
		t.Fatal("execution reset the process event cursor", err)
	}
	seen := false
	for _, event := range batch.Events {
		if event.Task == accepted.Response.Task {
			seen = true
			if event.Session != accepted.Response.Session {
				t.Fatal("event attributed to selected session", event.Session)
			}
		}
	}
	if !seen {
		t.Fatal("execution events unavailable")
	}
}
