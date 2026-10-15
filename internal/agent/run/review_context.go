// Purpose: Fulfil bounded review context requests through the same scoped read adapter.
// Pattern: Internal read use case.
// Usage: Conclave may request one follow-up context round; shell and mutation are unavailable.
// Related: internal/agent/run/conclave.go, internal/tools/local/prepare.go.
package run

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) reviewContext(ctx context.Context, needed conclave.ContextRequest) (json.RawMessage, error) {
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{"path": needed.Path, "offset": needed.Offset, "limit": needed.Limit})
	if err != nil {
		return nil, err
	}
	call := toolapi.Call{ID: "review_" + rand.Text(), Name: "read_file", Arguments: body}
	record, err := grounding.Bind(call, needed.Reason, s.checkpoint.ID)
	if err != nil {
		return nil, err
	}
	if err := s.record(ctx, Event{Kind: "self_grounding", Grounding: &record, Internal: true}); err != nil {
		return nil, err
	}
	op, err := s.runner.Tools.Prepare(ctx, s.runner.Settings, true, call)
	if err != nil {
		return nil, err
	}
	if op.Approval().Required {
		return nil, fmt.Errorf("review context requires an unexpected approval")
	}
	result, err := op.Execute(ctx, false)
	event := Event{Kind: "tool_succeeded", CallID: call.ID, Call: &call, Output: result.Content, Text: result.Summary, Internal: true}
	if err != nil || result.Failed {
		event.Kind = "tool_failed"
	}
	if recordErr := s.record(ctx, event); recordErr != nil {
		return nil, recordErr
	}
	if err != nil {
		return nil, err
	}
	if result.Failed {
		return nil, fmt.Errorf("review context read failed: %s", result.Summary)
	}
	return result.Content, nil
}
