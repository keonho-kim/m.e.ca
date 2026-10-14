// Purpose: Verify Plan toggles persist cancellation and cannot revive late worker drafts.
// Pattern: Terminal and persistence integration test.
// Usage: go test ./internal/tui/session -run TestPlanToggle.
// Related: internal/tui/session/plan_mode.go, internal/tui/session/worker_result.go.
package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
)

func enterPlanCommand(a *app) {
	a.editor.Set("!plan")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
}

func TestPlanTogglePersistsOff(t *testing.T) {
	a := fixture(t)
	enterPlanCommand(a)
	first := a.plan.ID
	enterPlanCommand(a)
	if a.planMode || a.busy || a.plan.ApprovedAt != nil {
		t.Fatal("second !plan did not cancel without execution")
	}
	if err := a.restore(); err != nil {
		t.Fatal(err)
	}
	if a.planMode {
		t.Fatal("cancelled Plan was restored as active")
	}
	enterPlanCommand(a)
	if !a.planMode || a.plan.ID == first {
		t.Fatal("new Plan did not start independently")
	}
}

func TestPlanToggleWaitsForWorkerBeforeSavingOff(t *testing.T) {
	a := fixture(t)
	enterPlanCommand(a)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.busy, a.taskCancel = true, cancel
	enterPlanCommand(a)
	if ctx.Err() == nil || !a.busy {
		t.Fatal("active planner was not cancelled and awaited")
	}
	draft, err := planning.RevisePlan(*a.plan, "late result", []string{"read files"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.opts.Store.SavePlan(a.ctx, draft); err != nil {
		t.Fatal(err)
	}
	a.receive(notice{Done: true, Plan: &draft, Err: context.Canceled})
	if a.planMode || a.busy || a.page != "chat" {
		t.Fatal("late draft revived Plan mode")
	}
	if err := a.restore(); err != nil {
		t.Fatal(err)
	}
	if a.planMode {
		t.Fatal("late persisted revision revived Plan mode")
	}
}

type rejectingPlanStore struct{ Store }

func (rejectingPlanStore) SavePlan(context.Context, planning.Plan) error {
	return errors.New("save failed")
}

func TestPlanToggleKeepsReadOnlyOnSaveFailure(t *testing.T) {
	a := fixture(t)
	enterPlanCommand(a)
	a.opts.Store = rejectingPlanStore{a.opts.Store}
	enterPlanCommand(a)
	if !a.planMode || a.plan.Status != planning.Planning {
		t.Fatal("failed save released the read-only mode")
	}
}
