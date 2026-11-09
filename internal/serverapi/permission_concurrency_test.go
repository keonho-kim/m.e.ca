// Purpose: Reject global permission changes while another native session owns an active task.
// Pattern: Controlled task-lifecycle integration test.
// Usage: go test -race ./internal/serverapi -run TestGlobalPermission.
// Related: internal/commands/server/settings.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"testing"
)

func TestGlobalPermissionCannotChangeWhileAnotherSessionRuns(t *testing.T) {
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		<-ctx.Done()
		return ctx.Err()
	}), false, 2)
	task, cursor := startNativeTask(t, f, "permission-task", "wait")
	if _, err := f.app.Select(deadline(t), "", true); err != nil {
		t.Fatal(err)
	}
	permission := "default"
	if _, err := f.app.Configure(deadline(t), f.app.State().Session.ID, &permission, nil); faults.Code(err) != diagcatalog.Conflict {
		t.Fatal("global policy changed around an active task", err)
	}
	if err := f.app.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	waitNativeTask(t, deadline(t), f.app, task.ID, cursor)
	if _, err := f.app.Configure(deadline(t), f.app.State().Session.ID, &permission, nil); err != nil {
		t.Fatal("idle policy update rejected", err)
	}
}
