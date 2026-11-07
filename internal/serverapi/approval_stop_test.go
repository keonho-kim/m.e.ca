// Purpose: Ensure denial and cancellation release pending approval without executing its effect.
// Pattern: Approval failure-path integration tests.
// Usage: go test ./internal/serverapi -run TestPendingApprovalStops.
// Related: internal/serverapi/fixture_test.go, internal/commands/server/controls.go.
package serverapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestPendingApprovalStopsWithoutEffect(t *testing.T) {
	for _, action := range []string{"deny", "cancel"} {
		t.Run(action, func(t *testing.T) {
			ctx := deadline(t)
			f := newFixture(t, generateFunc(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
				return emit(model.Event{Calls: []tool.Call{{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"unapproved.txt","content":"must not run"}`)}}, Done: true})
			}), true)
			state, err := f.app.Select(ctx, "", true)
			if err != nil {
				t.Fatal(err)
			}
			task, err := f.app.Start(ctx, state.Session.ID, "request-stop", "write file")
			if err != nil {
				t.Fatal(err)
			}
			cursor, approval := state.Cursor, ""
			for approval == "" {
				batch, err := f.app.Next(ctx, cursor)
				if err != nil {
					t.Fatal(err)
				}
				for _, e := range batch.Events {
					cursor = e.Cursor
					if e.Record.Kind == "approval_required" {
						approval = e.ApprovalID
					}
				}
			}
			if action == "deny" {
				err = f.app.Approve(task.ID, approval, false)
			} else {
				err = f.app.Cancel(task.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			finished := waitFinished(t, ctx, f.app, cursor)
			if finished.Status == "answered" || f.executed.Load() != 0 {
				t.Fatal(finished, f.executed.Load())
			}
			if err := f.app.Approve(task.ID, approval, true); err == nil {
				t.Fatal("stale approval was accepted")
			}
			if _, err := os.Stat(filepath.Join(f.project, "unapproved.txt")); !os.IsNotExist(err) {
				t.Fatal("unapproved effect", err)
			}
		})
	}
}
