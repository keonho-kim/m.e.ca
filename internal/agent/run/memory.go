// Purpose: Refresh complete conversation context at safe compaction boundaries and account new tokens.
// Pattern: Compaction integration use case.
// Usage: Called by the task loop; compaction shares the same model-call and time budget.
// Related: internal/agent/compact/controller.go, internal/agent/run/generate.go, internal/agent/run/context.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *task) observe(ctx context.Context, count compact.TokenCount) error {
	if s.runner.Memory == nil {
		return nil
	}
	return s.runner.Memory.Observe(ctx, count)
}
func (s *task) refreshMemory(ctx context.Context) error {
	if s.runner.Memory == nil {
		var err error
		s.messages, err = s.runner.Journal.ContextMessages(ctx, s.runner.Session, 0, s.limits.MaxContextBytes)
		return err
	}
	changed, err := s.runner.Memory.Refresh(ctx, func(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if s.usage.ModelCalls >= s.limits.MaxModelCalls {
			return fmt.Errorf("model-call budget exhausted during compaction")
		}
		s.usage.ModelCalls++
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		request.Thinking = s.runner.Thinking
		request.ReasoningEffort = s.runner.ReasoningEffort
		return s.generate(ctx, request, emit)
	})
	if err != nil {
		return err
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
