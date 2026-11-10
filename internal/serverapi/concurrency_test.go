// Purpose: Keep native tasks isolated across sessions and enforce the configured execution capacity.
// Pattern: Channel-controlled engine and durable-ledger integration test.
// Usage: go test -race ./internal/serverapi -run TestConcurrentTasks.
// Related: internal/commands/server/tasks.go, internal/commands/server/task_registry.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"testing"
)

func TestConcurrentTasksStayIndependentAndReleaseCapacity(t *testing.T) {
	ctx := deadline(t)
	started := make(chan struct{}, 3)
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}), false, 2)
	first, cursor := startNativeTask(t, f, "first-task", "first")
	second, _ := startNativeTask(t, f, "second-task", "second")
	for range 2 {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if first.Session == second.Session {
		t.Fatal("independent tasks share a session")
	}
	state, err := f.app.Select(ctx, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.Start(ctx, state.Session.ID, "third-task", "third"); faults.Code(err) != diagcatalog.ResourceExhausted {
		t.Fatal("capacity did not reject admission", err)
	}
	if err := f.app.Cancel(first.ID); err != nil {
		t.Fatal(err)
	}
	waitNativeTask(t, ctx, f.app, first.ID, cursor)
	if task, err := f.app.TaskStatus(ctx, second.ID); err != nil || task.Status != "running" {
		t.Fatal("one cancellation changed another task", task, err)
	}
	third, err := f.app.Start(ctx, state.Session.ID, "third-task", "third")
	if err != nil {
		t.Fatal("completion did not release capacity", err)
	}
	for _, id := range []string{second.ID, third.ID} {
		if err := f.app.Cancel(id); err != nil {
			t.Fatal(err)
		}
	}
}
