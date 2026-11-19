// Purpose: Count request occupancy for compacting history using reported usage, measured input or bytes.
// Pattern: Usage accounting boundary.
// Usage: generate counts agent attempts; compaction generation uses a separate context.
// Related: internal/agent/run/usage.go, internal/agent/compact/usage.go.
package run

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func (s *execution) countRequest(ctx context.Context, request modelapi.Request, usage *modelapi.TokenUsage, inputTokens int, inputBytes, outputBytes int64) error {
	if s.runner.Memory == nil {
		return nil
	}
	count := compact.TokenCount{Kind: "request"}
	if usage != nil {
		count.Amount = usage.Input + usage.Output
	} else if inputTokens > 0 {
		count.Amount = int64(inputTokens) + compact.EstimateBytes(outputBytes).Amount
		count.Estimated = outputBytes > 0
	} else {
		if inputBytes <= 0 {
			inputBytes = modelapi.TextBytes(request)
		}
		count = compact.EstimateBytes(inputBytes + outputBytes)
	}
	return s.runner.Memory.Observe(ctx, count)
}
