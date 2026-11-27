// Purpose: Own code execution, its native dialogue pair and final model-facing result.
// Pattern: Request-owned orchestration with nested durable effects.
// Usage: The decision loop dispatches only validated codemode calls here.
// Related: internal/agent/run/code_dispatch.go, internal/agent/run/code_result.go.
package run

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/source"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/redaction"
	"github.com/keonho-kim/meca/internal/tools/codemode"
)

var errCodeBoundary = faults.New(diag.FailedPrecondition, diag.CodeModelBoundary)

type codeObservation struct {
	Tool   string          `json:"tool"`
	Value  json.RawMessage `json:"value"`
	Source *source.Public  `json:"source,omitempty"`
}

type codeExecution struct {
	call                toolapi.Call
	messageID           string
	replies             map[string]codemode.Reply
	observations        []codeObservation
	images              []media.Image
	boundary            bool
	terminal            error
	calls               int
	instructionRevision uint64
	invalidArguments    bool
	transient           bool
	stable              bool
	observedThrough     int
	lastFailure         *codeObservation
}

func (s *execution) codeActions(ctx context.Context, reply modelapi.Response, calls []validatedCall, observedThrough int, approve Approver) error {
	reply.Calls = make([]toolapi.Call, len(calls))
	for i := range calls {
		reply.Calls[i] = calls[i].call
	}
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState, Calls: reply.Calls, PreserveObservations: true}); err != nil {
		return err
	}
	if reply.Text != "" {
		if err := s.emit(runrecord.Event{Kind: "progress", Text: reply.Text}); err != nil {
			return err
		}
	}
	for i, call := range calls {
		err := s.executeCode(ctx, call.call, observedThrough, approve)
		if err != nil || s.result.Departed {
			skippedCause := err
			if skippedCause == nil {
				skippedCause = errCodeBoundary
			}
			if errors.Is(err, errCodeBoundary) {
				err = nil
			}
			for _, remaining := range calls[i+1:] {
				state := &codeExecution{call: remaining.call, messageID: "code_" + rand.Text()}
				err = errors.Join(err, s.finishCode(ctx, state, nil, "not_executed", skippedCause))
			}
			return err
		}
	}
	return nil
}

func (s *execution) executeCode(ctx context.Context, call toolapi.Call, observedThrough int, approve Approver) error {
	state := &codeExecution{call: call, messageID: "code_" + rand.Text(), replies: make(map[string]codemode.Reply), instructionRevision: s.instructionRevision, observedThrough: observedThrough}
	s.code = state
	defer func() { s.code = nil }()
	s.checkpoint.Tool, s.checkpoint.Phase = codemode.Name, "execution"
	started := runrecord.Event{Kind: "tool_started", CallID: call.ID, Call: &call, Text: "Running code"}
	if err := s.persist(ctx, started); err != nil {
		return errors.Join(err, s.finishCode(ctx, state, nil, "not_executed", err))
	}
	if err := s.emit(started); err != nil {
		return errors.Join(err, s.finishCode(ctx, state, nil, "not_executed", err))
	}
	s.statistics.codeExecutions++
	var input struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(call.Arguments, &input); err != nil {
		return errors.Join(err, s.finishCode(ctx, state, nil, "not_executed", err))
	}
	names := make([]string, len(s.definitions))
	for i, definition := range s.definitions {
		names[i] = definition.Name
	}
	codeCtx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	result, err := codemode.Run(codeCtx, input.Code, names, func(ctx context.Context, calls []codemode.Call, deliver codemode.Deliver) error {
		return s.dispatchCode(ctx, calls, approve, deliver)
	})
	state.calls = result.Calls
	status := "completed"
	if err != nil {
		status = "error"
	}
	if errors.Is(err, errCodeBoundary) {
		status, err = "needs_model", nil
	}
	if s.result.Departed && err == nil {
		status = "completed"
	}
	if saveErr := s.finishCode(ctx, state, result.Value, status, err); saveErr != nil {
		return errors.Join(err, saveErr, state.terminal)
	}
	if state.terminal != nil {
		return errors.Join(state.terminal, fatalBatchError(ctx, err), ctx.Err())
	}
	if state.invalidArguments && err != nil {
		s.codeArgumentCorrections++
		if s.codeArgumentCorrections > maxArgumentCorrections {
			return faults.New(diag.FailedPrecondition, diag.RunToolArgumentCorrectionExhausted, maxArgumentCorrections, s.runner.Redact(err.Error()))
		}
	} else {
		s.codeArgumentCorrections = 0
	}
	if state.boundary {
		if err != nil {
			return err
		}
		return errCodeBoundary
	}
	return errors.Join(fatalBatchError(ctx, err), state.terminal, ctx.Err())
}

func (s *execution) finishCode(ctx context.Context, state *codeExecution, value json.RawMessage, status string, cause error) error {
	if len(value) == 0 {
		value = json.RawMessage(`null`)
	}
	value, err := redaction.RedactJSON(value, s.runner.Redact)
	if err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	var failure string
	if cause != nil {
		failure = excerpt(s.runner.Redact(cause.Error()), codemode.MaxErrorBytes)
		s.result.HadToolFailure = true
	}
	body, err := json.Marshal(struct {
		Status          string            `json:"status"`
		Result          json.RawMessage   `json:"result"`
		Error           string            `json:"error,omitempty"`
		Calls           int               `json:"tool_calls"`
		Observations    []codeObservation `json:"observations,omitempty"`
		LastToolFailure *codeObservation  `json:"last_tool_failure,omitempty"`
	}{status, value, failure, state.calls, state.observations, state.lastFailure})
	if err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	kind := "tool_succeeded"
	if cause != nil {
		kind = "tool_failed"
	}
	event := runrecord.Event{Kind: kind, Call: &state.call, CallID: state.call.ID, Output: body, Text: "Code " + status}
	if cause != nil {
		event.ErrorCode = faults.Code(cause)
		event.Text = failure
	}
	checkpoint, err := s.checkpointEvent()
	if err != nil {
		return err
	}
	message := modelapi.Message{ID: state.messageID, Role: "tool", ToolCallID: state.call.ID, ToolName: codemode.Name, Content: string(body), Images: state.images}
	message.TransientObservation = state.transient && !state.stable && cause == nil
	message.AcknowledgesInput = cause == nil
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.save(recordCtx, message, event, checkpoint); err != nil {
		return err
	}
	if message.AcknowledgesInput {
		s.acknowledgeObservations(state.observedThrough)
	}
	if err := s.emit(event); err != nil && !(ctx.Err() != nil && errors.Is(err, ctx.Err())) {
		return errors.Join(errRuntimeRecord, err)
	}
	return nil
}
