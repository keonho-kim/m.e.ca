// Purpose: Commit and acknowledge unavailable command captures before stopping their owning request.
// Pattern: Journal fault and command lifecycle integration test.
// Usage: go test ./internal/agent/run -run TestCommandCaptureFailure.
// Related: internal/agent/run/commands.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type unavailableCaptureFixture struct{ *commandFixture }

func (*unavailableCaptureFixture) CommandCapture(context.Context, string) (json.RawMessage, error) {
	return nil, errors.New("injected missing capture")
}

type missingCaptureExecutor struct{ *local.Executor }

func (*missingCaptureExecutor) CommandCapture(context.Context, string) (json.RawMessage, error) {
	return nil, errors.New("injected lost original")
}

type unavailableCaptureBatch struct {
	*unavailableCaptureFixture
	acknowledged []string
}

func (f *unavailableCaptureBatch) CollectCommands() ([]tool.CommandCompletion, error) {
	first, err := f.commandFixture.CollectCommands()
	if err != nil || len(first) == 0 {
		return first, err
	}
	second := first[0]
	second.ID, second.CallID = "cmd_second", "call_second"
	process := *second.Result.Process
	process.CommandID = second.ID
	second.Result.Process = &process
	return append(first, second), nil
}
func (f *unavailableCaptureBatch) AcknowledgeCommand(id string) error {
	f.acknowledged = append(f.acknowledged, id)
	if len(f.acknowledged) == 2 {
		return f.commandFixture.AcknowledgeCommand(id)
	}
	return nil
}

func TestCommandCaptureFailureDrainsEveryCompletedCommand(t *testing.T) {
	j := &journal{}
	base := &commandFixture{executor: &executor{}, completed: true}
	commands := &unavailableCaptureBatch{unavailableCaptureFixture: &unavailableCaptureFixture{base}}
	r := fixture(&generator{}, base.executor, j)
	r.Tools = commands
	s := execution{runner: r, limits: DefaultLimits(), checkpoint: record.Checkpoint{ID: "run_batch_failure", Status: record.Running, Phase: "decision"}, emit: emit}
	_, err := s.collectCommands(t.Context())
	if err == nil || len(commands.acknowledged) != 2 || base.completed {
		t.Fatal("a first capture failure stranded another completed command", commands.acknowledged, err)
	}
}

func TestCommandCaptureFailureImmediateOutcomeRemainsDurable(t *testing.T) {
	calls := []tool.Call{batchCall("missing", "execute_command", map[string]any{"command": "printf harmless"})}
	r, tools, _, journal, _ := realBatchRunner(t, calls)
	r.Tools = &missingCaptureExecutor{tools}
	_, err := r.Run(t.Context(), "Inspect output", approve, emit)
	if err == nil || tools.HasCommands() {
		t.Fatal("capture failure was hidden or retained the completed process", err)
	}
	found := false
	for _, event := range journal.events {
		if event.Kind == "tool_failed" && event.CallID == "missing" && event.Operation != nil && event.Operation.Failure != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("failed capture lost its durable tool outcome")
	}
}

func TestCommandCaptureFailureIsAcknowledgedOnlyAfterCommit(t *testing.T) {
	for _, failCommit := range []bool{false, true} {
		t.Run(map[bool]string{false: "recorded", true: "journal-failed"}[failCommit], func(t *testing.T) {
			j := &journal{}
			if failCommit {
				j.failKind = "command_finished"
			}
			base := &commandFixture{executor: &executor{}, completed: true}
			r := fixture(&generator{}, base.executor, j)
			r.Tools = &unavailableCaptureFixture{base}
			s := execution{runner: r, limits: DefaultLimits(), checkpoint: record.Checkpoint{ID: "run_failure", Status: record.Running, Phase: "decision"}, emit: emit}
			_, err := s.collectCommands(t.Context())
			if err == nil {
				t.Fatal("missing capture did not stop the request")
			}
			if base.completed != failCommit {
				t.Fatalf("acknowledgement crossed durability: completed=%v commitFailure=%v", base.completed, failCommit)
			}
			if !failCommit && len(j.events) == 0 {
				t.Fatal("failure provenance was not recorded")
			}
		})
	}
}
