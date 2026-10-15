// Purpose: Apply shared serial budgets and actual usage accounting to a review call.
// Pattern: Model I/O adapter.
// Usage: Conclave invokes reviewGenerate once, with at most one bounded follow-up.
// Related: internal/agent/run/usage.go, internal/agent/run/conclave.go.
package run

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *execution) reviewGenerate(ctx context.Context, request modelapi.Request, consume func(modelapi.Event) error) error {
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		return fmt.Errorf("model call budget exhausted during review")
	}
	s.usage.ModelCalls++
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	size := 0
	return s.generate(ctx, request, true, func(e modelapi.Event) error {
		if e.Retry != nil {
			size = 0
		}
		size += len(e.Delta) + len(e.Thinking)
		if size > modelapi.MaxResponseBytes {
			return fmt.Errorf("review exceeds model response bound")
		}
		return consume(e)
	})
}
