// Purpose: Freeze, approve and recheck a tool group before any operation starts.
// Pattern: Serial admission with bounded preparation workers.
// Usage: The code scheduler owns successful prepared operations and their cleanup.
// Related: internal/agent/run/code_schedule.go, internal/agent/run/batch_prepare.go.
package run

import (
	"context"
	"errors"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *execution) prepareWave(ctx context.Context, active, calls []validatedCall, preparation int, approve Approver) ([]*readyExecution, error) {
	revision := s.instructionRevision
	ready, index, outcome, err := s.freezeWave(ctx, calls, preparation)
	if err != nil {
		return nil, errors.Join(s.rejectWave(ctx, calls, index, outcome, err), fatalBatchError(ctx, err))
	}
	admitted := false
	defer func() {
		if !admitted {
			releaseReady(ready)
		}
	}()
	for i, prepared := range ready {
		approved, outcome, err := s.approveExecution(ctx, prepared, approve, s.emit)
		if err != nil {
			return nil, errors.Join(s.rejectWave(ctx, calls, i, outcome, err), fatalBatchError(ctx, err))
		}
		ready[i] = approved
	}
	if err := s.checkSteering(ctx, revision); err != nil {
		var failure *faults.Failure
		if errors.As(err, &failure) && failure.Code == diag.InstructionChanged {
			return nil, s.skipCalls(ctx, calls, skipForInstruction, failure.Detail)
		}
		return nil, errors.Join(err, s.skipCalls(ctx, calls, skipForContext, err.Error()))
	}
	commandsOnly := true
	for _, call := range calls {
		commandsOnly = commandsOnly && call.call.Name == "execute_command"
	}
	if !commandsOnly {
		if plan, err := s.planWave(ctx, active, calls); err != nil || plan.Count != len(calls) {
			detail := "Tool targets changed or overlap after preparation; nothing in this group executed."
			if err != nil {
				detail += " " + err.Error()
			}
			return nil, errors.Join(s.skipCalls(ctx, calls, skipAfterFailure, detail), fatalBatchError(ctx, err))
		}
	}
	admitted = true
	return ready, nil
}
