// Purpose: Attribute detached native task events to their own session after foreground selection changes.
// Pattern: Native task/session ownership integration test.
// Usage: go test ./internal/serverapi -run TestNativeSelectionDoesNotReassignTask.
// Related: internal/commands/server/lifecycle.go, internal/commands/server/events.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	"testing"
)

func TestNativeSelectionDoesNotReassignTask(t *testing.T) {
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		<-ctx.Done()
		return ctx.Err()
	}), false)
	task, cursor := startNativeTask(t, f, "owned-session-task", "wait")
	selected, err := f.app.Select(deadline(t), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Session.ID == task.Session || selected.Execution.Session.ID != task.Session {
		t.Fatal("selection changed task ownership", selected)
	}
	if err := f.app.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	waitNativeTask(t, deadline(t), f.app, task.ID, cursor)
	batch, err := f.app.Replay(cursor)
	if err != nil || batch.Snapshot != nil {
		t.Fatal(err)
	}
	seen := false
	for _, event := range batch.Events {
		if event.Task == task.ID {
			seen = true
			if event.Session != task.Session {
				t.Fatal("task event used foreground session", event)
			}
		}
	}
	if !seen {
		t.Fatal("task events lost after selection changed")
	}
}
