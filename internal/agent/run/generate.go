// Purpose: Own one bounded model reply and preserve interrupted output without exposing argument drafts.
// Pattern: Model I/O boundary.
// Usage: The main decision loop receives replies under the request call and time budget.
// Related: internal/agent/run/context.go, internal/agent/run/run.go, internal/agent/model/model.go.
package run

import (
	"context"
	"errors"
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

func (s *execution) askResponse(ctx context.Context, request modelapi.Request, send modelapi.Stream, draft *replyDraft) (modelapi.Response, error) {
	if err := s.checkTimeBudget(ctx); err != nil {
		return modelapi.Response{}, err
	}
	if !s.modelCall() {
		return modelapi.Response{}, errModelBudget
	}
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	s.presentation = draft.presentation
	hideThinking := s.runner.HideThinking
	var reply, thinking strings.Builder
	var maskReply, maskThinking func(string, bool) string
	var replay replyReplay
	var retained, attemptBytes int64
	beginAttempt := func() error {
		reply.Reset()
		thinking.Reset()
		maskReply, maskThinking = s.streamMask(), nil
		if !hideThinking {
			maskThinking = s.streamMask()
		}
		retained = draft.output.bytes()
		attemptBytes = 0
		replay = replyReplay{catchingUp: draft.output.visible(hideThinking) != (replyOutput{})}
		if s.replyMemory != nil {
			return s.replyMemory.Resize(retained)
		}
		return nil
	}
	if err := beginAttempt(); err != nil {
		return modelapi.Response{}, err
	}
	snapshot := func(final bool) replyOutput {
		output := replyOutput{text: maskReply(reply.String(), final)}
		if final && maskThinking == nil {
			maskThinking = s.streamMask()
		}
		if maskThinking != nil {
			output.thinking = maskThinking(thinking.String(), final)
		}
		return output
	}
	capture := func(output replyOutput, final bool) error {
		previous, next := draft.output.visible(hideThinking), output.visible(hideThinking)
		if output == (replyOutput{}) || !replay.accepts(previous, next) || !final && previous == next {
			return nil
		}
		draft.output = output
		retained = 0
		if previous == next || ctx.Err() != nil {
			return nil
		}
		return s.emit(runrecord.Event{Kind: "stream", Text: next.text, Thinking: next.thinking})
	}
	var calls []toolapi.Call
	var continuation []byte
	var inputBudget modelapi.InputBudget
	var measuredUsage modelapi.TokenUsage
	var wireBytes int64
	done := false
	err := s.generate(ctx, request, inference.Agent, send, func(e modelapi.Event) error {
		if e.InputBytes > 0 {
			wireBytes = e.InputBytes
		}
		if e.InputBudget != nil {
			inputBudget = *e.InputBudget
		}
		if e.Usage != nil {
			measuredUsage = *e.Usage
		}
		if e.Retry != nil {
			if err := capture(snapshot(true), true); err != nil {
				return err
			}
			measuredUsage = modelapi.TokenUsage{}
			calls, continuation, done = nil, nil, false
			return beginAttempt()
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
			if err := s.replyMemory.Resize(retained + int64(size)); err != nil {
				return err
			}
		}
		attemptBytes = int64(size)
		// Check before allocating append buffers or masking an oversized event.
		reply.WriteString(e.Delta)
		thinking.WriteString(e.Thinking)
		if e.Done {
			done = true
		}
		if e.Streaming && (e.Delta != "" || !hideThinking && e.Thinking != "") {
			if err := capture(snapshot(false), false); err != nil {
				return err
			}
		}
		return ctx.Err()
	})
	draft.presentation = s.presentation
	output := snapshot(true)
	if err == nil {
		if !done {
			err = faults.Errorf(diagcatalog.RunModelReplyDidNotComplete)
		} else {
			err = s.checkReply(reply.String(), continuation, calls)
		}
	}
	if err != nil {
		return modelapi.Response{}, errors.Join(err, capture(output, true))
	}
	if s.replyMemory != nil {
		// Completion releases the previous attempt's held snapshot.
		if err := s.replyMemory.Resize(attemptBytes); err != nil {
			return modelapi.Response{}, err
		}
	}
	clearDraft := len(calls) > 0 && output.text == "" && draft.output.visible(hideThinking) != (replyOutput{})
	*draft = replyDraft{}
	if clearDraft {
		if err := s.emit(runrecord.Event{Kind: "stream"}); err != nil {
			return modelapi.Response{}, err
		}
	}
	if err := s.recordReply(ctx, "", output.thinking, false); err != nil {
		return modelapi.Response{}, err
	}
	if output.thinking != "" && !hideThinking {
		if err := s.emit(runrecord.Event{Kind: "thinking", Thinking: output.thinking}); err != nil {
			return modelapi.Response{}, err
		}
	}
	response := modelapi.Response{Text: reply.String(), Reasoning: output.thinking, ReasoningState: continuation, Calls: calls}
	s.webFetchAllowance = s.pageAllowance(request, response, inputBudget, measuredUsage, wireBytes)
	return response, nil
}

func (s *execution) recordReply(ctx context.Context, text, thinking string, interrupted bool) error {
	if interrupted {
		thinking = s.runner.Redact(thinking)
	}
	if thinking == "" && (!interrupted || text == "") {
		return nil
	}
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	thought := runrecord.Event{Kind: "thinking", Thinking: thinking}
	if interrupted && text != "" {
		draft := modelapi.Message{Role: "assistant", Content: text, Incomplete: true}
		if thinking != "" {
			return s.save(flush, draft, thought)
		}
		return s.save(flush, draft)
	}
	return s.record(flush, thought)
}

func (s *execution) checkReply(text string, continuation []byte, calls []toolapi.Call) error {
	if s.runner.Redact(text) != text {
		return faults.Errorf(diagcatalog.RunModelReplyContainsAProtectedCredentialOr)
	}
	if err := redaction.CheckProtected(continuation, s.runner.Redact); err != nil {
		return err
	}
	ids := map[string]bool{}
	for _, call := range calls {
		if ids[call.ID] {
			return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.RunDuplicateCallIDsInOneResponse)}
		}
		ids[call.ID] = true
		if err := redaction.CheckProtected(call.Arguments, s.runner.Redact); err != nil {
			return err
		}
	}
	return nil
}

func (s *execution) streamMask() func(string, bool) string {
	if s.runner.NewStreamMask != nil {
		return s.runner.NewStreamMask()
	}
	return func(text string, _ bool) string {
		return s.runner.Redact(text)
	}
}
