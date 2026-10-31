// Purpose: Count a finished request attempt toward compaction using measured usage or byte fallback.
// Pattern: Usage accounting boundary.
// Usage: generate counts agent attempts; compaction and independent review use separate contexts.
// Related: internal/agent/run/usage.go, internal/agent/compact/usage.go.
package run

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func (s *execution) countRequest(ctx context.Context, request modelapi.Request, usage *modelapi.TokenUsage, inputBytes, outputBytes int64) error {
	if s.runner.Memory == nil {
		return nil
	}
	count := compact.TokenCount{Kind: "request"}
	if usage != nil {
		count.Amount = usage.Input + usage.Output
	} else {
		if inputBytes <= 0 {
			inputBytes = modelapi.TextBytes(request)
		}
		count = compact.EstimateBytes(inputBytes + outputBytes)
	}
	return s.observe(ctx, count)
}
