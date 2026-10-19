// Purpose: Validate one native tool call and orchestrate approval and execution.
// Pattern: Native tool execution use case.
// Usage: Invoked once per complete native tool call, before the existing approval/execution boundary.
// Related: internal/agent/tool/arguments.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"time"
)

func (s *execution) action(ctx context.Context, reply modelapi.Response, approve Approver) error {
	s.usage.recordTools(1)
	v, err := s.validateCall(reply.Calls[0])
	if err != nil {
		return s.returnToolFeedback(ctx, reply, err.Error())
	}
	call, definition, arguments := v.call, v.definition, v.arguments
	s.checkpoint.Iteration = s.usage.snapshot().ToolCalls
	s.checkpoint.Tool = definition.Name
	s.checkpoint.Arguments = arguments
	s.checkpoint.Phase = "approval"
	if err := s.persist(ctx); err != nil {
		return err
	}
	// The compiled call enters the audit transcript only after every argument is valid.
	if err = s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, Calls: reply.Calls}); err != nil {
		return err
	}
	if reply.Text != "" {
		if err := s.emit(Event{Kind: "progress", Text: reply.Text}); err != nil {
			return err
		}
	}
	if err := s.record(ctx, Event{Kind: "tool_proposed", CallID: call.ID, Call: &call, Text: "Native candidate; not executed"}); err != nil {
		return err
	}
	outcome, toolErr := s.execute(ctx, call, approve, s.emit)
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(s.acceptResult(recordCtx, v, outcome, toolErr), ctx.Err())
}
