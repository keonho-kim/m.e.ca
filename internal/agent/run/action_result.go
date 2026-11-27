// Purpose: Commit native effects independently of the enclosing model-facing code result.
// Pattern: Ordered durable result application.
// Usage: Single and batch execution update facts and record every actual inner outcome.
// Related: internal/agent/run/code_result.go, internal/agent/run/observations.go.
package run

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"errors"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func (s *execution) acceptResult(ctx context.Context, v validatedCall, outcome toolapi.Result, toolErr error, events ...runrecord.Event) error {
	failed := outcome.Failed
	if errors.Is(toolErr, errRuntimeRecord) {
		if outcome.Started {
			return errors.Join(toolErr, s.observeResult(v, outcome))
		}
		return toolErr
	}
	if outcome.Depart && toolErr == nil && !failed {
		s.result.Departed = true
	}
	s.result.HadToolFailure = s.result.HadToolFailure || failed || toolErr != nil
	if ctx.Err() != nil {
		return errors.Join(ctx.Err(), s.observeResult(v, outcome))
	}
	if toolErr != nil || failed {
		failure := outcome.Failure
		if failure == nil {
			failure = faults.FailureOf(toolErr)
		}
		if failure == nil {
			failure = &faults.Failure{Code: diagcatalog.ExecutionFailed, Detail: diagcatalog.Text(diagcatalog.RunToolReportedFailure), Next: diagcatalog.Text(diagcatalog.RunInspectItsOutputBeforeChoosingAnotherAction)}
		}
		failure.Detail = s.runner.Redact(failure.Detail)
		outcome.Failure = failure
	}

	if err := s.observeResult(v, outcome); err != nil {
		return err
	}
	s.checkpoint.Phase = "observation"
	s.checkpoint.Observation = s.runner.Redact(outcome.Summary)
	if outcome.Started {
		s.checkpoint.ChangeRevision++
	}
	if outcome.Failure != nil {
		s.checkpoint.Blocker = s.runner.Redact(outcome.Failure.Detail)
	} else {
		s.checkpoint.Blocker = ""
	}
	event, err := s.checkpointEvent()
	if err != nil {
		return err
	}
	events = append(events, event)
	if err := s.recordEvents(ctx, events); err != nil {
		return err
	}
	return s.acceptCodeResult(v, outcome, toolErr)
}
