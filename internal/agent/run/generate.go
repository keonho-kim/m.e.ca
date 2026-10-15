// Purpose: Own one bounded model reply and preserve interrupted output without exposing argument drafts.
// Pattern: Model I/O boundary.
// Usage: Called by decision and internal model phases under a shared call and time budget.
// Related: internal/agent/run/context.go, internal/agent/run/run.go, internal/agent/tool/arguments.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
)

func (s *execution) askResponse(ctx context.Context, question string, role inference.Role, maxBytes int) (modelapi.Response, error) {
	if err := ctx.Err(); err != nil {
		return modelapi.Response{}, err
	}
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		return modelapi.Response{}, fmt.Errorf("model call budget exhausted")
	}
	request, err := s.request(question, role)
	if err != nil {
		return modelapi.Response{}, err
	}
	s.usage.ModelCalls++
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	if err := s.emit(Event{Kind: "stream"}); err != nil {
		return modelapi.Response{}, err
	}
	holdDraft := len(s.checkpoint.Work.Receipts) > 0 && s.checkpoint.Phase == "decision" || s.checkpoint.Phase == "reporting"
	var reply, thinking strings.Builder
	var calls []toolapi.Call
	done := false
	err = s.generate(ctx, request, true, func(e modelapi.Event) error {
		if e.Retry != nil {
			reply.Reset()
			calls = nil
			thinking.Reset()
			done = false
			return s.emit(Event{Kind: "stream"})
		}
		if len(e.Calls) > 0 {
			if len(request.Tools) == 0 || !e.Done || len(calls) != 0 || len(e.Calls) > toolapi.MaxCalls {
				return &modelapi.ToolCallError{Detail: "unexpected or incomplete tool calls in this phase"}
			}
			calls = append(calls, e.Calls...)
		}
		reply.WriteString(e.Delta)
		thinking.WriteString(e.Thinking)
		size := reply.Len() + thinking.Len()
		for _, call := range calls {
			size += len(call.Arguments) + len(call.ID) + len(call.Name)
		}
		if reply.Len() > maxBytes || size > modelapi.MaxResponseBytes {
			return &modelapi.OutputLimitError{Reason: "current phase byte limit"}
		}
		if e.Done {
			done = true
		}
		if !s.runner.suppressStream && e.Streaming && (e.Delta != "" || e.Thinking != "") {
			visible := maskedPartial(s.runner, s.responseBody(reply.String()))
			if holdDraft {
				visible = ""
			}
			if err := s.emit(Event{Kind: "stream", Text: visible, Thinking: maskedPartial(s.runner, thinking.String())}); err != nil {
				return err
			}
		}
		return ctx.Err()
	})
	if err == nil && !done {
		err = fmt.Errorf("model reply did not complete")
	}
	if thinking.Len() > 0 {
		event := Event{Kind: "thinking", Internal: s.runner.suppressStream, Thinking: maskedPartial(s.runner, thinking.String())}
		if saveErr := s.record(context.WithoutCancel(ctx), event); saveErr != nil {
			return modelapi.Response{}, saveErr
		}
		if err == nil && !s.runner.suppressStream {
			if emitErr := s.emit(event); emitErr != nil {
				return modelapi.Response{}, emitErr
			}
		}
	}
	if err != nil {
		if thinking.Len() > 0 && !s.runner.suppressStream {
			archived := modelapi.Message{Role: "system", Content: "Interrupted assistant thinking (unfinished, unverified historical reasoning):", Reasoning: maskedPartial(s.runner, thinking.String())}
			if saveErr := s.save(context.WithoutCancel(ctx), archived); saveErr != nil {
				return modelapi.Response{}, saveErr
			}
		}
		if body := s.responseBody(reply.String()); body != "" {
			if saveErr := s.save(context.WithoutCancel(ctx), modelapi.Message{Role: "assistant", Content: maskedPartial(s.runner, body)}); saveErr != nil {
				return modelapi.Response{}, saveErr
			}
		}
		return modelapi.Response{}, err
	}
	if s.runner.Redact(reply.String()) != reply.String() {
		return modelapi.Response{}, fmt.Errorf("model reply contains a protected credential or terminal control")
	}
	ids := map[string]bool{}
	for _, call := range calls {
		if ids[call.ID] {
			return modelapi.Response{}, &modelapi.ToolCallError{Detail: "duplicate call IDs in one response"}
		}
		ids[call.ID] = true
		if s.runner.Redact(string(call.Arguments)) != string(call.Arguments) {
			return modelapi.Response{}, fmt.Errorf("tool arguments contain a protected credential or terminal control")
		}
	}
	return modelapi.Response{Text: reply.String(), Reasoning: maskedPartial(s.runner, thinking.String()), Calls: calls}, nil
}

func maskedPartial(r Runner, text string) string {
	if r.MaskStream != nil {
		return r.MaskStream(text)
	}
	return r.Redact(text)
}
