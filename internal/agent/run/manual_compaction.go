// Purpose: Compact an idle conversation with the normal model budget and usage accounting.
// Pattern: Summary-only execution use case.
// Usage: Session dispatch calls CompactNow for the explicit !compact command.
// Related: internal/agent/run/memory.go, internal/agent/session/request.go, internal/agent/compact/controller.go.
package run

import (
	"context"
	"time"

	"github.com/keonho-kim/meca/internal/agent/presentation"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (r Runner) CompactNow(ctx context.Context, emit func(runrecord.Event) error) (result Result, err error) {
	if r.Memory == nil || r.Generator == nil || r.Journal == nil || r.Redact == nil || emit == nil {
		return result, fault.Errorf(diag.CompactCompactionDependenciesAreRequired)
	}
	limits, err := r.checkedLimits()
	if err != nil {
		return result, err
	}
	ctx, _ = presentation.Scope(ctx)
	ctx = diagnostics.WithRedactor(ctx, r.Redact)
	release := r.scopeGenerator()
	defer release()
	started := time.Now()
	s := r.newExecution(ctx, limits, emit, started)
	ctx = diagnostics.WithFields(ctx, "component", "agent", "session_id", r.Session, "run_id", s.checkpoint.ID)
	ctx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	diagnostics.Log(ctx, diag.RunStarted)
	defer func() {
		if recover() != nil {
			err = fault.Panic()
		}
		s.finishState(err)
		result = s.result
		diagnostics.Log(ctx, diag.RunFinished, "status", string(result.Status), "duration_ms", time.Since(started).Milliseconds(), "error_code", diagnostics.Code(err))
	}()
	changed, err := r.Memory.CompactNow(ctx, func() error {
		return s.emit(runrecord.Event{Kind: "compaction_started"})
	}, s.generateCompaction)
	if err != nil {
		return result, err
	}
	if !changed {
		return result, fault.New(diag.FailedPrecondition, diag.ManualCompactionEmpty)
	}
	completion := runrecord.Event{Kind: "compaction_finished", Presentation: s.order.Next()}
	flush, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	if err := s.record(flush, completion); err != nil {
		return result, err
	}
	s.checkpoint.Status = runrecord.Answered
	return result, s.emit(completion)
}
