// Purpose: Own one bounded model reply and preserve interrupted output without exposing argument drafts.
// Pattern: Model I/O boundary.
// Usage: The main decision loop receives replies under the request call and time budget.
// Related: internal/agent/run/context.go, internal/agent/run/run.go, internal/agent/model/model.go.
package run

import (
	"context"

	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/redaction"
)

func (s *execution) askResponse(ctx context.Context) (modelapi.Response, error) {
	if err := ctx.Err(); err != nil {
		return modelapi.Response{}, err
	}
	if !s.modelCall() {
		return modelapi.Response{}, errModelBudget
	}
	if s.runner.BeforeDecision != nil {
		s.runner.BeforeDecision()
	}
	request, err := s.request()
	if err != nil {
		return modelapi.Response{}, err
	}
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	if err := s.emit(runrecord.Event{Kind: "stream"}); err != nil {
		return modelapi.Response{}, err
	}
	if s.replyMemory != nil {
		if err := s.replyMemory.Resize(0); err != nil {
			return modelapi.Response{}, err
		}
	}
	var reply, thinking strings.Builder
	maskReply := s.streamMask()
	var maskThinking func(string, bool) string
	if !s.runner.HideThinking {
		maskThinking = s.streamMask()
	}
	var visibleReply, visibleThinking string
	var calls []toolapi.Call
	var continuation []byte
	done := false
	err = s.generate(ctx, request, inference.Agent, func(e modelapi.Event) error {
		if e.Retry != nil {
			reply.Reset()
			calls = nil
			continuation = nil
			thinking.Reset()
			maskReply = s.streamMask()
			if !s.runner.HideThinking {
				maskThinking = s.streamMask()
			}
			visibleReply, visibleThinking = "", ""
			if s.replyMemory != nil {
				if err := s.replyMemory.Resize(0); err != nil {
					return err
				}
			}
			done = false
			return s.emit(runrecord.Event{Kind: "stream"})
		}
		if len(e.Calls) > 0 {
			if len(request.Tools) == 0 || !e.Done || len(calls) != 0 || len(e.Calls) > toolapi.MaxCalls {
				return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.RunUnavailableIncompleteOrExcessiveNativeToolCalls)}
			}
			calls = append(calls, e.Calls...)
		}
		if len(e.ReasoningState) > 0 {
			if !e.Done {
				return faults.Errorf(diagcatalog.RunProviderContinuationStateArrivedBeforeCompletion)
			}
			continuation = e.ReasoningState
		}
		size := reply.Len() + thinking.Len() + len(continuation) + len(e.Delta) + len(e.Thinking)
		for _, call := range calls {
			size += len(call.Arguments) + len(call.ID) + len(call.Name)
		}
		if size > modelapi.MaxResponseBytes {
			return &modelapi.OutputLimitError{Reason: "model response byte limit"}
		}
		if s.replyMemory != nil {
			if err := s.replyMemory.Resize(int64(size)); err != nil {
				return err
			}
		}
		// Check before allocating append buffers or masking an oversized event.
		reply.WriteString(e.Delta)
		thinking.WriteString(e.Thinking)
		visibleReply = maskReply(reply.String(), false)
		if maskThinking != nil {
			visibleThinking = maskThinking(thinking.String(), false)
		}
		if e.Done {
			done = true
		}
		if e.Streaming && (e.Delta != "" || !s.runner.HideThinking && e.Thinking != "") {
			if err := s.emit(runrecord.Event{Kind: "stream", Text: visibleReply, Thinking: visibleThinking}); err != nil {
				return err
			}
		}
		return ctx.Err()
	})
	if maskThinking == nil {
		maskThinking = s.streamMask()
	}
	visibleReply, visibleThinking = maskReply(reply.String(), true), maskThinking(thinking.String(), true)
	if err == nil && !done {
		err = faults.Errorf(diagcatalog.RunModelReplyDidNotComplete)
	}
	if thinking.Len() > 0 {
		event := runrecord.Event{Kind: "thinking", Thinking: visibleThinking}
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		saveErr := s.record(flush, event)
		cancel()
		if saveErr != nil {
			return modelapi.Response{}, saveErr
		}
		if err == nil {
			if !s.runner.HideThinking {
				if emitErr := s.emit(event); emitErr != nil {
					return modelapi.Response{}, emitErr
				}
			}
		}
	}
	if err != nil {
		// The thinking event preserves the visible trace without replaying an
		// unfinished reasoning chain as input to the next generation.
		if body := reply.String(); body != "" {
			flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			saveErr := s.save(flush, modelapi.Message{Role: "assistant", Content: visibleReply, Incomplete: true})
			cancel()
			if saveErr != nil {
				return modelapi.Response{}, saveErr
			}
		}
		return modelapi.Response{}, err
	}
	if s.runner.Redact(reply.String()) != reply.String() {
		return modelapi.Response{}, faults.Errorf(diagcatalog.RunModelReplyContainsAProtectedCredentialOr)
	}
	if err := redaction.CheckProtected(continuation, s.runner.Redact); err != nil {
		return modelapi.Response{}, err
	}
	ids := map[string]bool{}
	for _, call := range calls {
		if ids[call.ID] {
			return modelapi.Response{}, &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.RunDuplicateCallIDsInOneResponse)}
		}
		ids[call.ID] = true
		if err := redaction.CheckProtected(call.Arguments, s.runner.Redact); err != nil {
			return modelapi.Response{}, err
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
