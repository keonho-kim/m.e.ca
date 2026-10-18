// Purpose: Refresh complete conversation context at safe compaction boundaries and account new tokens.
// Pattern: Compaction integration use case.
// Usage: Called by the execution loop; compaction shares the same model-call and time budget.
// Related: internal/agent/compact/controller.go, internal/agent/run/generate.go, internal/agent/run/context.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
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
			resultErr = errors.Join(resultErr, s.emit(Event{Kind: "compaction_finished"}))
		}
	}()
	if s.runner.Memory == nil {
		if force {
			return fmt.Errorf("context limit reached without an available compaction store")
		}
		var err error
		s.messages, err = s.runner.Journal.ContextMessages(ctx, s.runner.Session, 0, s.limits.MaxContextBytes)
		return err
	}
	s.runner.Memory.ActiveRequest = s.checkpoint.Objective
	s.runner.Memory.ActiveApproach = s.workBackground().ProposedApproach
	refresh := s.runner.Memory.Refresh
	if force {
		refresh = s.runner.Memory.Compact
	}
	changed, err := refresh(ctx, func(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if !s.modelCall() {
			return fmt.Errorf("compaction: %w", errModelBudget)
		}
		if !compressing {
			if err := s.emit(Event{Kind: "compaction_started"}); err != nil {
				return err
			}
			compressing = true
		}
		s.checkpoint.ModelCalls = s.usage.snapshot().ModelCalls
		request.Thinking = s.runner.Thinking
		request.ReasoningEffort = s.runner.ReasoningEffort
		parameters, err := inference.Resolve(s.runner.Inference, inference.Compaction)
		if err != nil {
			return err
		}
		request = parameters.ApplyThinking(request, s.runner.ThinkingSupported)
		return s.generate(ctx, request, false, emit)
	})
	if err != nil {
		return err
	}
	if force && !changed {
		return fmt.Errorf("context limit reached but no older history can be compacted; reduce fixed instructions or increase model context")
	}
	var through int64
	s.memoryContext, through, err = s.runner.Memory.Context()
	if err != nil {
		return err
	}
	s.messages, err = s.runner.Journal.ContextMessages(ctx, s.runner.Session, through, s.limits.MaxContextBytes)
	if err != nil {
		return err
	}
	if changed {
		event := Event{Kind: "compaction", Text: "Compacted conversation memory; original records retained"}
		if err = s.record(ctx, event); err != nil {
			return err
		}
		return s.emit(event)
	}
	return nil
}
