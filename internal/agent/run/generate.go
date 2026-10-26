// Purpose: Own one bounded model reply and preserve interrupted output without exposing argument drafts.
// Pattern: Model I/O boundary.
// Usage: The main decision loop receives replies under the request call and time budget.
// Related: internal/agent/run/context.go, internal/agent/run/run.go, internal/agent/model/model.go.
package run

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
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
	if err := s.emit(runrecord.Event{Kind: "stream"}); err != nil {
		return modelapi.Response{}, err
	}
	var reply, thinking strings.Builder
	maskReply, maskThinking := s.streamMask(), s.streamMask()
	var visibleReply, visibleThinking string
	var calls []toolapi.Call
	var continuation []byte
	done := false
	err = s.generate(ctx, request, true, func(e modelapi.Event) error {
		if e.Retry != nil {
			reply.Reset()
			calls = nil
			continuation = nil
			thinking.Reset()
			maskReply, maskThinking = s.streamMask(), s.streamMask()
			visibleReply, visibleThinking = "", ""
			done = false
			return s.emit(runrecord.Event{Kind: "stream"})
		}
		if len(e.Calls) > 0 {
			if len(request.Tools) == 0 || !e.Done || len(calls) != 0 || len(e.Calls) > toolapi.MaxCalls {
				return &modelapi.ToolCallError{Detail: "unavailable, incomplete or excessive native tool calls"}
			}
			calls = append(calls, e.Calls...)
		}
		reply.WriteString(e.Delta)
		thinking.WriteString(e.Thinking)
		visibleReply = maskReply(reply.String(), false)
		visibleThinking = maskThinking(thinking.String(), false)
		if len(e.ReasoningState) > 0 {
			if !e.Done {
				return fmt.Errorf("provider continuation state arrived before completion")
			}
			continuation = e.ReasoningState
		}
		size := reply.Len() + thinking.Len() + len(continuation)
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
			if err := s.emit(runrecord.Event{Kind: "stream", Text: visibleReply, Thinking: visibleThinking}); err != nil {
				return err
			}
		}
		return ctx.Err()
	})
	visibleReply, visibleThinking = maskReply(reply.String(), true), maskThinking(thinking.String(), true)
	if err == nil && !done {
		err = fmt.Errorf("model reply did not complete")
	}
	if thinking.Len() > 0 {
		event := runrecord.Event{Kind: "thinking", Thinking: visibleThinking}
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
		// The thinking event preserves the visible trace without replaying an
		// unfinished reasoning chain as input to the next generation.
		if body := reply.String(); body != "" {
			if saveErr := s.save(context.WithoutCancel(ctx), modelapi.Message{Role: "assistant", Content: visibleReply, Incomplete: true}); saveErr != nil {
				return modelapi.Response{}, saveErr
			}
		}
		return modelapi.Response{}, err
	}
	if s.runner.Redact(reply.String()) != reply.String() {
		return modelapi.Response{}, fmt.Errorf("model reply contains a protected credential or terminal control")
	}
	if s.runner.Redact(string(continuation)) != string(continuation) {
		return modelapi.Response{}, fmt.Errorf("provider continuation state contains protected content")
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
	return modelapi.Response{Text: reply.String(), Reasoning: visibleThinking, ReasoningState: continuation, Calls: calls}, nil
}

func (s *execution) streamMask() func(string, bool) string {
	if s.runner.NewStreamMask != nil {
		return s.runner.NewStreamMask()
	}
	return func(text string, _ bool) string {
		return s.runner.Redact(text)
	}
}
