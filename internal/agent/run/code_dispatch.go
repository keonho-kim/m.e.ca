// Purpose: Deliver committed tool completions while independent native operations continue.
// Pattern: Single-owner event loop with bounded workers and joined cancellation.
// Usage: Code Mode calls this dispatcher; workers only execute prepared operations.
// Related: internal/agent/run/code_admission.go, internal/agent/run/code_schedule.go.
package run

import (
	"context"
	"errors"
	"sync"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
	"github.com/keonho-kim/meca/internal/tools/codemode"
)

type codeTask struct {
	id    int
	call  validatedCall
	ready *readyExecution
}
type codeCompletion struct {
	task    *codeTask
	outcome toolapi.Result
	err     error
}
type codeDispatcher struct {
	execution                 *execution
	ctx                       context.Context
	approve                   Approver
	pending, active, feedback []*codeTask
	completed                 chan codeCompletion
	workers                   sync.WaitGroup
}

func (s *execution) dispatchCode(ctx context.Context, inputs []codemode.Call, approve Approver, deliver codemode.Deliver) (err error) {
	ctx = instruction.WithExplicitReads(ctx, s.explicitSkills)
	ctx, cancel := context.WithCancel(ctx)
	d := &codeDispatcher{execution: s, ctx: ctx, approve: approve, completed: make(chan codeCompletion, maxParallelTools)}
	defer func() {
		cancel()
		// Effects already started must be joined and committed even after JS fails.
		for len(d.active) > 0 {
			completion := <-d.completed
			_, recordErr := d.record(completion)
			if errors.Is(recordErr, errRuntimeRecord) || faults.IsPanic(recordErr) {
				err = errors.Join(err, recordErr)
			}
		}
		d.workers.Wait()
		if len(d.pending) > 0 {
			reason := skipAfterFailure
			if s.result.Departed {
				reason = skipForClosure
			} else if s.code.boundary {
				reason = skipForInstruction
			} else if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				reason = skipForInterruption
			}
			err = errors.Join(err, d.skipPending(reason, "Code execution stopped before these operations started."))
		}
		clear(s.code.replies)
		err = errors.Join(err, s.code.terminal)
	}()
	if err = d.admit(inputs); err != nil {
		return err
	}
	for len(d.pending)+len(d.active)+len(d.feedback) > 0 {
		if err = d.check(); err != nil {
			return err
		}
		if _, err = d.drain(); err != nil {
			return err
		}
		if len(d.feedback) > 0 {
			task := d.feedback[0]
			d.feedback[0] = nil
			d.feedback = d.feedback[1:]
			reply, ok := s.code.replies[task.call.call.ExecutionKey()]
			if !ok {
				return errors.Join(errRuntimeRecord, faults.New(diag.Internal, diag.CodeDispatchMismatch))
			}
			delete(s.code.replies, task.call.call.ExecutionKey())
			inputs, deliveryErr := deliver(task.id, reply)
			if deliveryErr != nil {
				return deliveryErr
			}
			if err = d.admit(inputs); err != nil {
				return err
			}
			continue
		}
		if len(d.pending) > 0 && len(d.active) < maxParallelTools {
			started, startErr := d.start()
			if startErr != nil {
				return startErr
			}
			if started {
				continue
			}
		}
		if len(d.active) == 0 {
			return faults.Errorf(diag.RunInvalidParallelBatchBoundary)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case completion := <-d.completed:
			if _, err = d.accept(completion); err != nil {
				return err
			}
		}
	}
	return d.check()
}

func (d *codeDispatcher) accept(completion codeCompletion) (bool, error) {
	failed, err := d.record(completion)
	if terminal := errors.Join(fatalBatchError(d.ctx, err), d.execution.code.terminal); terminal != nil {
		return failed, terminal
	}
	d.feedback = append(d.feedback, completion.task)
	if failed {
		err = d.skipPending(skipAfterFailure, "An earlier tool failed; these pending calls did not execute. Inspect completed results before retrying.")
	}
	return failed, err
}

func (d *codeDispatcher) drain() (bool, error) {
	failed := false
	for {
		select {
		case completion := <-d.completed:
			current, err := d.accept(completion)
			failed = failed || current
			if err != nil {
				return failed, err
			}
		default:
			return failed, d.check()
		}
	}
}

func (d *codeDispatcher) check() error {
	s := d.execution
	if s.code.terminal != nil {
		return s.code.terminal
	}
	if s.code.boundary || s.result.Departed {
		return errCodeBoundary
	}
	if err := s.checkTimeBudget(d.ctx); err != nil {
		return err
	}
	return nil
}

func (d *codeDispatcher) checkInstructions() error {
	s := d.execution
	if err := s.checkSteering(d.ctx, s.code.instructionRevision); err != nil {
		if failure := faults.FailureOf(err); failure != nil && failure.Code == diag.InstructionChanged {
			s.code.boundary = true
			return errCodeBoundary
		}
		return err
	}
	return nil
}

func (d *codeDispatcher) record(completion codeCompletion) (bool, error) {
	task := completion.task
	outcome, err := d.execution.recordExecution(d.ctx, task.call, task.ready, completion.outcome, completion.err, d.execution.emit)
	toolapi.Release(task.ready.operation)
	task.ready = nil
	for i, active := range d.active {
		if active == task {
			copy(d.active[i:], d.active[i+1:])
			d.active[len(d.active)-1] = nil
			d.active = d.active[:len(d.active)-1]
			break
		}
	}
	return outcome.Failed || err != nil, err
}

func (d *codeDispatcher) skipPending(reason skipReason, detail string) error {
	pending := d.pending
	d.pending = nil // An uncertain journal write must not be retried by cleanup.
	calls := codeCalls(pending)
	if len(calls) == 0 {
		return nil
	}
	if err := d.execution.skipCalls(d.ctx, calls, reason, detail); err != nil {
		return err
	}
	d.feedback = append(d.feedback, pending...)
	clear(pending)
	return nil
}

func codeCalls(tasks []*codeTask) []validatedCall {
	calls := make([]validatedCall, len(tasks))
	for i, task := range tasks {
		calls[i] = task.call
	}
	return calls
}
