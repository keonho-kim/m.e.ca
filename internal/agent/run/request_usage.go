// Purpose: Count a finished request attempt toward compaction using measured usage or byte fallback.
// Pattern: Usage accounting boundary.
// Usage: generate finalizes each attempt once; compaction requests opt out of this counter.
// Related: internal/agent/run/usage.go, internal/agent/compact/usage.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) countRequest(ctx context.Context, request modelapi.Request, usage *modelapi.TokenUsage, outputBytes int64) error {
	if s.runner.Memory == nil {
		return nil
	}
	count := compact.TokenCount{Kind: "request"}
	if usage != nil {
		count.Amount = usage.Input + usage.Output
	} else {
		messages := append([]modelapi.Message(nil), request.Messages...)
		for i := range messages {
			messages[i].Images = nil
		}
		body, err := json.Marshal(struct {
			Messages []modelapi.Message
			Tools    []toolapi.Definition
		}{messages, request.Tools})
		if err != nil {
			return err
		}
		count = compact.EstimateBytes(int64(len(body)) + outputBytes)
	}
	return s.observe(ctx, count)
}
