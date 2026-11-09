// Purpose: Refresh complete conversation context at safe compaction boundaries and account new tokens.
// Pattern: Compaction integration use case.
// Usage: Called by the execution loop; compaction shares the same model-call and time budget.
// Related: internal/agent/compact/controller.go, internal/agent/run/generate.go, internal/agent/run/context.go.
package run

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/compact"
	contextbuild "github.com/keonho-kim/meca/internal/agent/context"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *execution) observe(ctx context.Context, count compact.TokenCount) error {
	if s.runner.Memory == nil {
		return nil
	}
	return s.runner.Memory.Observe(ctx, count)
}
func (s *execution) refreshMemory(ctx context.Context, force bool) (resultErr error) {
	compressing := false
	defer func() {
		if compressing {
			resultErr = errors.Join(resultErr, s.emit(runrecord.Event{Kind: "compaction_finished"}))
		}
	}()
	if s.runner.Memory == nil {
		if force {
			return faults.Errorf(diagcatalog.RunContextLimitReachedWithoutAnAvailableCompaction)
		}
		var err error
		if !s.historyLoaded {
			err = s.loadContext(ctx, 0)
			s.historyLoaded = err == nil
		}
		return err
	}
	s.runner.Memory.ActiveRequest = s.checkpoint.Objective
	refresh := s.runner.Memory.Refresh
	if force {
		refresh = s.runner.Memory.Compact
	}
	started := func() error {
		if err := s.emit(runrecord.Event{Kind: "compaction_started"}); err != nil {
			return err
		}
		compressing = true
		return nil
	}
	changed, err := refresh(ctx, started, func(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if !s.modelCall() {
			return faults.Errorf(diagcatalog.RunCompaction, errModelBudget)
		}
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		request.Thinking = s.runner.Thinking
		request.ReasoningEffort = s.runner.ReasoningEffort
		parameters, err := inference.Resolve(s.runner.Inference, inference.Compaction)
		if err != nil {
			return err
		}
		request = parameters.ApplyThinking(request, s.runner.ThinkingSupported)
		return s.generate(ctx, request, inference.Compaction, emit)
	})
	if err != nil {
		return err
	}
	if force && !changed {
		return faults.Errorf(diagcatalog.RunContextLimitReachedButNoOlderHistory)
	}
	var through int64
	s.memoryContext, through = s.runner.Memory.Context()
	if changed || !s.historyLoaded {
		s.projection = contextbuild.Projection{}
		err = s.loadContext(ctx, through)
		if err != nil {
			return err
		}
		s.historyLoaded = true
	}
	if changed {
		event := runrecord.Event{Kind: "compaction", Text: "Compacted conversation memory; original records retained"}
		if err = s.record(ctx, event); err != nil {
			return err
		}
		return s.emit(event)
	}
	return nil
}
