// Purpose: Own one bounded model reply and preserve interrupted output without exposing argument drafts.
// Pattern: Model I/O boundary.
// Usage: The main decision loop receives replies under the request call and time budget.
// Related: internal/agent/run/context.go, internal/agent/run/run.go, internal/agent/tool/arguments.go.
package run

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
)

func (s *execution) askResponse(ctx context.Context) (modelapi.Response, error) {
	if err := ctx.Err(); err != nil {
		return modelapi.Response{}, err
	}
	if !s.modelCall() {
		return modelapi.Response{}, errModelBudget
	}
	request, err := s.request()
	if err != nil {
		return modelapi.Response{}, err
	}
	s.checkpoint.ModelCalls = s.usage.snapshot().ModelCalls
	if err := s.emit(Event{Kind: "stream"}); err != nil {
		return modelapi.Response{}, err
	}
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
				return &modelapi.ToolCallError{Detail: "unavailable, incomplete or excessive native tool calls"}
			}
			calls = append(calls, e.Calls...)
		}
		reply.WriteString(e.Delta)
		thinking.WriteString(e.Thinking)
		size := reply.Len() + thinking.Len()
		for _, call := range calls {
			size += len(call.Arguments) + len(call.ID) + len(call.Name)
		}
		if size > modelapi.MaxResponseBytes {
			return &modelapi.OutputLimitError{Reason: "model response byte limit"}
		}
		if e.Done {
			done = true
		}
		if e.Streaming && (e.Delta != "" || e.Thinking != "") {
			visible := maskedPartial(s.runner, reply.String())
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
		event := Event{Kind: "thinking", Thinking: maskedPartial(s.runner, thinking.String())}
		if saveErr := s.record(context.WithoutCancel(ctx), event); saveErr != nil {
			return modelapi.Response{}, saveErr
		}
		if err == nil {
			if emitErr := s.emit(event); emitErr != nil {
				return modelapi.Response{}, emitErr
			}
		}
	}
	if err != nil {
		if thinking.Len() > 0 {
			archived := modelapi.Message{Role: "system", Content: "Earlier unfinished notes:", Reasoning: maskedPartial(s.runner, thinking.String())}
			if saveErr := s.save(context.WithoutCancel(ctx), archived); saveErr != nil {
				return modelapi.Response{}, saveErr
			}
		}
		if body := reply.String(); body != "" {
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
