// Purpose: Own one bounded model reply and preserve interrupted output without exposing argument drafts.
// Pattern: Model I/O boundary.
// Usage: Called by decision and argument phases under a shared call and time budget.
// Related: internal/agent/run/context.go, internal/agent/run/run.go, internal/agent/pfc/pfc.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
)

func (s *task) ask(ctx context.Context, question string, role inference.Role, maxBytes int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		return "", fmt.Errorf("model call budget exhausted")
	}
	request, err := s.request(question, role)
	if err != nil {
		return "", err
	}
	s.usage.ModelCalls++
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	if err := s.emit(Event{Kind: "stream"}); err != nil {
		return "", err
	}
	var reply, thinking strings.Builder
	done := false
	outputTokens := 0
	err = s.generate(ctx, request, func(e modelapi.Event) error {
		if e.Retry != nil {
			reply.Reset()
			thinking.Reset()
			done, outputTokens = false, 0
			return s.emit(Event{Kind: "stream"})
		}
		if len(e.Calls) > 0 {
			return fmt.Errorf("native JSON tool calls are not accepted; use the staged text dialogue")
		}
		reply.WriteString(e.Delta)
		thinking.WriteString(e.Thinking)
		if reply.Len() > maxBytes || reply.Len()+thinking.Len() > modelapi.MaxResponseBytes {
			return &modelapi.OutputLimitError{Reason: "current phase byte limit"}
		}
		if e.Done {
			done = true
			if e.Usage != nil {
				outputTokens = int(e.Usage.Output)
			}
		}
		if !s.runner.suppressStream && e.Streaming && (e.Delta != "" || e.Thinking != "") {
			if err := s.emit(Event{Kind: "stream", Text: maskedPartial(s.runner, s.responseBody(reply.String())), Thinking: maskedPartial(s.runner, thinking.String())}); err != nil {
				return err
			}
		}
		return ctx.Err()
	})
	if err == nil && !done {
		err = fmt.Errorf("model reply did not complete")
	}
	count := compact.Estimate(reply.String()+thinking.String(), "model")
	if outputTokens > 0 {
		count = compact.TokenCount{Amount: int64(outputTokens), Kind: "model"}
	}
	if countErr := s.observe(context.WithoutCancel(ctx), count); countErr != nil {
		return "", countErr
	}
	if thinking.Len() > 0 {
		event := Event{Kind: "thinking", Internal: s.runner.suppressStream, Thinking: maskedPartial(s.runner, thinking.String())}
		if saveErr := s.record(context.WithoutCancel(ctx), event); saveErr != nil {
			return "", saveErr
		}
		if err == nil && !s.runner.suppressStream {
			if emitErr := s.emit(event); emitErr != nil {
				return "", emitErr
			}
		}
	}
	if err != nil {
		if body := s.responseBody(reply.String()); body != "" {
			if saveErr := s.save(context.WithoutCancel(ctx), modelapi.Message{Role: "assistant", Content: maskedPartial(s.runner, body)}); saveErr != nil {
				return "", saveErr
			}
		}
		return "", err
	}
	if s.runner.Redact(reply.String()) != reply.String() {
		return "", fmt.Errorf("model reply contains a protected credential or terminal control")
	}
	return reply.String(), nil
}

func maskedPartial(r Runner, text string) string {
	if r.MaskStream != nil {
		return r.MaskStream(text)
	}
	return r.Redact(text)
}
