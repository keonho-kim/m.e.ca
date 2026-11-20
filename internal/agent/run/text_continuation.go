// Purpose: Prepare one measured text-only continuation of a truncated answer.
// Pattern: Decision-owned bounded recovery.
// Usage: drive admits it after a provider token limit; askResponse joins only validated text.
// Related: internal/agent/run/decision_request.go, internal/agent/run/generate.go.
package run

import (
	"context"
	"errors"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type textContinuation struct {
	text     string
	revision uint64
	cause    error
}

func (c *textContinuation) prepare(s *execution, r *modelapi.Request) error {
	if c.revision != s.instructionRevision || s.runner.Redact(c.text) != c.text {
		return c.cause
	}
	// Borrow the already masked draft. Message roles keep model output separate from the instruction.
	r.Tools = nil
	r.Messages = append(r.Messages, modelapi.Message{Role: "assistant", Content: c.text},
		modelapi.Message{Role: "user", Content: "Continue the unfinished answer with only the missing text. Preserve whitespace at the boundary."})
	size := modelapi.TextBytes(*r)
	if size > int64(s.limits.MaxContextBytes) {
		return c.cause
	}
	return s.reserveContext(size)
}

func (c *textContinuation) fits(b modelapi.InputBudget, revision uint64) bool {
	return revision == c.revision && b.InputTokens > 0 && b.OutputTokens > 0 && b.InputTokens <= b.Window-b.OutputTokens
}

// join validates and reserves the combined size before allocating. Native state
// describes only the suffix call and cannot represent the combined assistant turn.
func (c *textContinuation) join(s *execution, draft *replyDraft, output replyOutput, complete bool) (replyOutput, error) {
	suffix := output.text
	if replay, ok := strings.CutPrefix(suffix, c.text); ok {
		suffix = replay
	} else if !complete && strings.HasPrefix(c.text, suffix) {
		suffix = ""
	}
	if suffix == "" {
		return draft.output, c.cause
	}
	if output.thinking == "" {
		output.thinking = draft.output.thinking
	}
	size := int64(len(c.text)) + int64(len(suffix)) + int64(len(output.thinking))
	if size > modelapi.MaxResponseBytes {
		return draft.output, c.cause
	}
	if s.replyMemory != nil {
		if err := s.replyMemory.Resize(size); err != nil {
			return draft.output, err
		}
	}
	output.text = c.text + suffix
	// A protected value can straddle the join even when both calls were individually safe.
	if err := s.checkReply(output.text, nil, nil); err != nil {
		return draft.output, err
	}
	return output, nil
}

func (c *textContinuation) finish(ctx context.Context, s *execution, draft *replyDraft, output replyOutput, cause error) (replyOutput, error) {
	if cause != nil && !textTokenLimit(cause) {
		return draft.output, cause
	}
	joined, err := c.join(s, draft, output, cause == nil)
	if err != nil {
		return draft.output, err
	}
	if cause != nil {
		draft.output = joined
		if ctx.Err() == nil {
			visible := joined.visible(s.runner.HideThinking)
			cause = errors.Join(cause, s.emit(runrecord.Event{Kind: "stream", Text: visible.text, Thinking: visible.thinking}))
		}
	}
	return joined, cause
}

func textTokenLimit(err error) bool {
	var limit *modelapi.OutputLimitError
	return errors.As(err, &limit) && limit.TokenLimit && !limit.PartialTool
}
