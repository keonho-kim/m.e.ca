// Purpose: Own one bounded model reply and preserve interrupted output without exposing argument drafts.
// Pattern: Model I/O boundary.
// Usage: Called by decision and argument phases under a shared call and time budget.
// Related: internal/agent/run/context.go, internal/agent/run/run.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"strings"
	"time"
)

func (s *task) ask(ctx context.Context, question string, maxOutput, maxBytes int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		return "", fmt.Errorf("model call budget exhausted")
	}
	request, err := s.request(question, maxOutput)
	if err != nil {
		return "", err
	}
	s.usage.ModelCalls++
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.limits.MaxModelSeconds)*time.Second)
	defer cancel()
	var reply, thinking strings.Builder
	done := false
	outputTokens := 0
	err = s.runner.Generator.Generate(ctx, request, func(e agent.ModelEvent) error {
		if len(e.Calls) > 0 {
			return fmt.Errorf("native JSON tool calls are not accepted; use the staged text dialogue")
		}
		reply.WriteString(e.Delta)
		thinking.WriteString(e.Thinking)
		if reply.Len() > maxBytes || reply.Len()+thinking.Len() > agent.MaxResponseBytes {
			return fmt.Errorf("model reply exceeds the current phase limit")
		}
		if e.Done {
			done = true
			outputTokens = e.OutputTokens
		}
		return ctx.Err()
	})
	count := compact.Estimate(reply.String()+thinking.String(), "model")
	if outputTokens > 0 {
		count = compact.TokenCount{Amount: int64(outputTokens), Kind: "model"}
	}
	if countErr := s.observe(context.WithoutCancel(ctx), count); countErr != nil {
		return "", countErr
	}
	if thinking.Len() > 0 {
		event := Event{Kind: "thinking", Thinking: maskedPartial(s.runner, thinking.String())}
		if saveErr := s.runner.Journal.RecordEvent(context.WithoutCancel(ctx), s.runner.Session, event); saveErr != nil {
			return "", saveErr
		}
		if err == nil {
			if emitErr := s.emit(event); emitErr != nil {
				return "", emitErr
			}
		}
	}
	if err != nil {
		if s.checkpoint.Phase == "decision" && reply.Len() > 0 {
			if saveErr := s.save(context.WithoutCancel(ctx), agent.Message{Role: "assistant", Content: maskedPartial(s.runner, reply.String())}); saveErr != nil {
				return "", saveErr
			}
		}
		return "", err
	}
	if !done {
		return "", fmt.Errorf("model reply did not complete")
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
