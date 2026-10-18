// Purpose: Preserve the final model report without requiring a verification command.
// Pattern: Execution completion transition.
// Usage: The single loop calls finish for a response without native calls.
// Related: internal/agent/run/run.go, internal/agent/run/finish_response.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"unicode/utf8"
)

func (s *execution) finish(ctx context.Context, text string) (bool, error) {
	text = strings.TrimSpace(text)
	if text == "" || !utf8.ValidString(text) {
		s.checkpoint.Observation = "Return a nonempty answer or one native call."
		return false, nil
	}
	s.checkpoint.Status = Answered
	s.result.Text = text
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: text}); err != nil {
		return false, err
	}
	return true, s.emit(Event{Kind: "response", Text: text})
}
