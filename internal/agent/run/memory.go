// Purpose: Account for newly added conversation tokens and compact only at a safe decision boundary.
// Pattern: Compaction integration use case.
// Usage: Called by the task loop; compaction shares the same model-call and time budget.
// Related: internal/agent/compact/controller.go, internal/agent/run/generate.go, internal/agent/run/context.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"time"
)

func (s *task) observe(ctx context.Context, count compact.TokenCount) error {
	if s.runner.Memory == nil {
		return nil
	}
	return s.runner.Memory.Observe(ctx, count)
}
func (s *task) refreshMemory(ctx context.Context) error {
	if s.runner.Memory == nil {
		return nil
	}
	changed, err := s.runner.Memory.Refresh(ctx, func(ctx context.Context, request agent.ModelRequest, emit func(agent.ModelEvent) error) error {
		if s.usage.ModelCalls >= s.limits.MaxModelCalls {
			return fmt.Errorf("model-call budget exhausted during compaction")
		}
		s.usage.ModelCalls++
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		ctx, cancel := context.WithTimeout(ctx, time.Duration(s.limits.MaxModelSeconds)*time.Second)
		defer cancel()
		request.Thinking = s.runner.Thinking
		return s.runner.Generator.Generate(ctx, request, emit)
	})
	if err != nil {
		return err
	}
	s.memoryContext, err = s.runner.Memory.Context()
	if err != nil {
		return err
	}
	if changed {
		event := Event{Kind: "compaction", Text: "Compacted conversation memory; original records retained"}
		if err = s.runner.Journal.RecordEvent(ctx, s.runner.Session, event); err != nil {
			return err
		}
		return s.emit(event)
	}
	return nil
}
