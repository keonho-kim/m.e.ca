// Purpose: Interrupt obsolete access waiters without releasing another command's workspace lease.
// Pattern: Generation-signal resource admission regressions.
// Usage: Workspace tests close an instruction generation at deterministic wait boundaries.
// Related: internal/tools/team/workspace/workspace.go, internal/agent/team/decision.go.
package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestNewInstructionInterruptsWaitWithoutReleasingCurrentShell(t *testing.T) {
	w := New()
	ctx := context.Background()
	shell, err := w.ReserveShell(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer shell.Release()
	obsolete := make(chan struct{})
	held, err := w.ReserveFiles(ctx, []string{"/project/file"}, obsolete, func() error {
		close(obsolete)
		return nil
	})
	var failure *tool.Failure
	if held != nil || !errors.As(err, &failure) || failure.Code != tool.InstructionChanged || ctx.Err() != nil {
		t.Fatal("waiting candidate did not return instruction feedback", err)
	}
	w.mu.Lock()
	stillHeld, writes := w.shell, len(w.writes)
	w.mu.Unlock()
	if !stillHeld || writes != 0 {
		t.Fatal("an obsolete waiter changed another owner's lease")
	}
	shell.Release()
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
	held, err := w.ReserveShell(context.Background(), obsolete, nil)
	var failure *tool.Failure
	if held != nil || !errors.As(err, &failure) || failure.Code != tool.InstructionChanged {
		t.Fatal("obsolete candidate acquired free access", err)
	}
}
