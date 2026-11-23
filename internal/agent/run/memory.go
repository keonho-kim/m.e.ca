// Purpose: Compact conversation memory at a decision boundary and reload the committed context.
// Pattern: Compaction integration use case.
// Usage: Called by the execution loop; compaction shares the same model-call and time budget.
// Related: internal/agent/compact/controller.go, internal/agent/run/generate.go, internal/agent/run/context.go.
package run

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *execution) compactMemory(ctx context.Context, budget modelapi.InputBudget, capacity *modelapi.ContextLimitError, discard func()) (resultErr error) {
	memory := s.runner.Memory
	if memory == nil {
		if capacity != nil {
			return faults.Errorf(diagcatalog.RunContextLimitReachedWithoutAnAvailableCompaction)
		}
		return nil
	}
	compressing := false
	var before modelapi.ContextMeasurement
	defer func() {
		if compressing {
			resultErr = errors.Join(resultErr, s.emit(runrecord.Event{Kind: "compaction_finished"}))
		}
	}()
	memory.ActiveRequest = s.checkpoint.Objective
	started := func(input modelapi.ContextMeasurement) error {
		before = input
		if discard != nil {
			discard()
		}
		if err := s.emit(runrecord.Event{Kind: "compaction_started"}); err != nil {
			return err
		}
		compressing = true
		return nil
	}
	var changed bool
	var err error
	if capacity != nil {
		changed, err = memory.Compact(ctx, *capacity, started, s.generateCompaction)
	} else {
		changed, err = memory.Refresh(ctx, budget, started, s.generateCompaction)
	}
	if err != nil {
		return err
	}
	if !changed {
		if capacity != nil {
			return faults.Errorf(diagcatalog.RunContextLimitReachedButNoOlderHistory)
		}
		return nil
	}
	var through int64
	s.memoryContext, through = memory.Context()
	if err := s.loadContext(ctx, through); err != nil {
		return err
	}
	s.historyLoaded = true
	s.beforeCompaction = &before
	event := runrecord.Event{Kind: "compaction", Text: "Compacted conversation memory; original records retained"}
	if err := s.record(ctx, event); err != nil {
		return err
	}
	return s.emit(event)
}

func (s *execution) generateCompaction(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	if err := s.modelCall(ctx); err != nil {
		return faults.Errorf(diagcatalog.RunCompaction, err)
	}
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	request.Thinking = s.runner.Thinking
	request.ReasoningEffort = s.runner.ReasoningEffort
	parameters, err := inference.Resolve(s.runner.Inference, inference.Compaction)
	if err != nil {
		return err
	}
	request = parameters.ApplyThinking(request, s.runner.ThinkingSupported)
	return s.generate(ctx, request, inference.Compaction, nil, emit)
}
