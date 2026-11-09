// Purpose: Interrupt obsolete access waiters without releasing another operation's target lease.
// Pattern: Generation-signal resource admission regressions.
// Usage: Workspace tests close an instruction generation at deterministic wait boundaries.
// Related: internal/tools/team/workspace/workspace.go, internal/agent/team/decision.go.
package workspace

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"errors"
	"testing"
)

func TestNewInstructionInterruptsWaitWithoutReleasingCurrentCommand(t *testing.T) {
	w := New()
	ctx := context.Background()
	command, err := w.ReserveCommand(ctx, []string{"/project/file"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer command.Release()
	obsolete := make(chan struct{})
	held, err := w.ReserveFiles(ctx, []string{"/project/file"}, obsolete, func() error {
		close(obsolete)
		return nil
	})
	var failure *faults.Failure
	if held != nil || !errors.As(err, &failure) || failure.Code != diagcatalog.InstructionChanged || ctx.Err() != nil {
		t.Fatal("waiting candidate did not return instruction feedback", err)
	}
	w.mu.Lock()
	_, stillHeld := w.writes[command]
	writes := len(w.writes)
	w.mu.Unlock()
	if !stillHeld || writes != 1 {
		t.Fatal("an obsolete waiter changed another owner's lease")
	}
	command.Release()
	current, err := w.ReserveFiles(ctx, []string{"/project/file"}, nil, nil)
	if err != nil {
		t.Fatal("a fresh candidate could not proceed", err)
	}
	current.Release()
}

func TestObsoleteCandidateCannotAcquireAnAlreadyFreeWorkspace(t *testing.T) {
	w := New()
	obsolete := make(chan struct{})
	close(obsolete)
	held, err := w.ReserveCommand(context.Background(), []string{"/project/file"}, obsolete, nil)
	var failure *faults.Failure
	if held != nil || !errors.As(err, &failure) || failure.Code != diagcatalog.InstructionChanged {
		t.Fatal("obsolete candidate acquired free access", err)
	}
}
