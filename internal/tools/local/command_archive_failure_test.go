// Purpose: Preserve a typed failed completion after archive failure without poisoning later requests.
// Pattern: Command lifecycle fault-injection regression.
// Usage: go test ./internal/tools/local -run TestCommandArchiveFailure.
// Related: internal/tools/local/commands.go, internal/tools/local/process/command_finish.go.
package local

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/tools/local/process"
)

func TestCommandArchiveFailureBecomesAcknowledgedCompletion(t *testing.T) {
	e, project := testExecutor(t)
	e.BeginCommands(t.Context(), time.Now().Add(time.Minute))
	e.commands.Save = func(string, string, process.Output, error) error { return errors.New("injected archive failure") }
	id, err := e.commands.Start(t.Context(), project, "printf harmless", os.Environ(), process.CallIdentity{CallID: "old", ExecutionID: "old_execution"})
	if err != nil {
		t.Fatal(err)
	}
	done, err := e.commands.Done(id)
	if err != nil {
		t.Fatal(err)
	}
	<-done
	completed, err := e.CollectCommands()
	if err != nil || len(completed) != 1 || !completed[0].Result.Failed || completed[0].Result.Failure == nil {
		t.Fatalf("archive failure must remain a typed collectible result: %+v %v", completed, err)
	}
	if completed[0].ExecutionID != "old_execution" || !completed[0].Result.Started {
		t.Fatal("archive failure lost execution provenance")
	}
	if err := e.AcknowledgeCommand(id); err != nil {
		t.Fatal(err)
	}
	e.StopCommands()
	e.commands.Save = e.archiveCommand
	e.BeginCommands(t.Context(), time.Now().Add(time.Minute))
	defer e.StopCommands()
	if results, err := e.CollectCommands(); err != nil || len(results) != 0 {
		t.Fatalf("old failure poisoned the next request: %+v %v", results, err)
	}
}
