// Purpose: Apply shared serial budgets and actual usage accounting to a review call.
// Pattern: Model I/O adapter.
// Usage: Conclave invokes reviewGenerate once, with at most one bounded follow-up.
// Related: internal/agent/run/usage.go, internal/agent/run/conclave.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *execution) reviewGenerate(ctx context.Context, request modelapi.Request, consume func(modelapi.Event) error) error {
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		return fmt.Errorf("model call budget exhausted during review")
	}
	s.usage.ModelCalls++
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	var usage *modelapi.TokenUsage
	size := 0
	err := s.generate(ctx, request, func(e modelapi.Event) error {
		if e.Retry != nil {
			size = 0
			usage = nil
		}
		size += len(e.Delta) + len(e.Thinking)
		if size > modelapi.MaxResponseBytes {
			return fmt.Errorf("review exceeds model response bound")
		}
		if e.Usage != nil {
			copy := *e.Usage
			usage = &copy
		}
		return consume(e)
	})
	count := compact.TokenCount{Amount: int64(size), Estimated: true, Kind: "model"}
	if usage != nil {
		count.Amount = usage.Output
		count.Estimated = false
	}
	return errors.Join(err, s.observe(context.WithoutCancel(ctx), count))
}
