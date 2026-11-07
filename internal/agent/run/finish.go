// Purpose: Preserve the final model report and its completed continuation state.
// Pattern: Execution completion transition.
// Usage: The single loop calls finish for a response without native calls.
// Related: internal/agent/run/run.go, internal/agent/model/model.go.
package run

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *execution) finish(ctx context.Context, reply modelapi.Response) error {
	text := strings.TrimSpace(reply.Text)
	if text == "" || !utf8.ValidString(text) {
		return fmt.Errorf("model completed with an empty or invalid response and no native call")
	}
	// Trimming a pathological padded reply can leave a tiny final value holding
	// megabytes alive in the result, history and presentation layers.
	if len(reply.Text) > 64<<10 && len(text) < len(reply.Text)/2 {
		text = strings.Clone(text)
	}
	s.checkpoint.Status = runrecord.Answered
	s.result.Text = text
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState}); err != nil {
		return err
	}
	return s.emit(runrecord.Event{Kind: "response", Text: text})
}
