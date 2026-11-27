// Purpose: Start bounded tool groups without overlapping active conflicting operations.
// Pattern: Native planning and approval on the serial dispatcher owner.
// Usage: Code dispatch starts workers only after admission, approval and durable intent.
// Related: internal/agent/run/batch.go, internal/agent/run/batch_wave.go.
package run

import "errors"

func (d *codeDispatcher) start() (bool, error) {
	s := d.execution
	calls, active := codeCalls(d.pending), codeCalls(d.active)
	plan, err := s.planWave(d.ctx, active, calls)
	if err != nil {
		return false, err
	}
	if plan.Count == 0 {
		return false, nil
	}
	if err := d.checkInstructions(); err != nil {
		return false, err
	}
	// Instructions and images are delivered to the model before further effects.
	for i := 0; i < plan.Count; i++ {
		if codeInputTool(calls[i].call.Name) {
			if i == 0 {
				if len(active) > 0 {
					return false, nil
				}
				plan.Count = 1
			} else {
				plan.Count = i
			}
			break
		}
	}
	plan.PreparationConcurrency = min(plan.PreparationConcurrency, plan.Count)
	tasks := append([]*codeTask(nil), d.pending[:plan.Count]...)
	clear(d.pending[:plan.Count])
	d.pending = d.pending[plan.Count:]
	calls = calls[:plan.Count]
	if len(calls) > 1 {
		batchID := "batch_" + calls[0].call.ExecutionID
		for i := range calls {
			calls[i].call.BatchID = batchID
			tasks[i].call = calls[i]
		}
	}
	ready, err := s.prepareWave(d.ctx, active, calls, plan.PreparationConcurrency, d.approve)
	if err != nil {
		return false, err
	}
	if len(ready) == 0 {
		d.feedback = append(d.feedback, tasks...)
		return true, d.skipPending(skipAfterFailure, "A tool group was rejected; these pending calls did not execute.")
	}
	started := false
	defer func() {
		if !started {
			releaseReady(ready)
		}
	}()
	// Approval can wait while earlier workers finish. Commit their outcomes
	// before admitting additional effects, including any terminal refusal.
	failed, err := d.drain()
	if err == nil {
		err = d.checkInstructions()
	}
	if failed || err != nil {
		reason := skipAfterFailure
		if s.result.Departed {
			reason = skipForClosure
		} else if s.code.boundary {
			reason = skipForInstruction
		}
		err = errors.Join(err, s.skipCalls(d.ctx, calls, reason, "An earlier operation or new instruction stopped this prepared group before execution."))
		d.feedback = append(d.feedback, tasks...)
		return true, err
	}
	if err := s.startExecutions(d.ctx, ready, s.emit); err != nil {
		return false, errors.Join(err, errRuntimeRecord, s.skipCalls(d.ctx, calls, skipForContext, err.Error()))
	}
	s.statistics.toolExecutions += int64(len(ready))
	started = true
	for i, task := range tasks {
		task.ready = ready[i]
		d.active = append(d.active, task)
		d.workers.Go(func() {
			outcome, err := executePrepared(d.ctx, task.ready)
			d.completed <- codeCompletion{task: task, outcome: outcome, err: err}
		})
	}
	return true, nil
}

func codeInputTool(name string) bool {
	return name == "load_skill" || name == "read_skill_file" || name == "view_image"
}
