// Purpose: Validate one native tool call and orchestrate approval and execution.
// Pattern: Native tool execution use case.
// Usage: Invoked once per complete native tool call, before the existing approval/execution boundary.
// Related: internal/agent/tool/arguments.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
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
	if err = s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState, Calls: reply.Calls}); err != nil {
		return err
	}
	if reply.Text != "" {
		if err := s.emit(runrecord.Event{Kind: "progress", Text: reply.Text}); err != nil {
			return err
		}
	}
	if err := s.record(ctx, runrecord.Event{Kind: "tool_proposed", CallID: call.ID, Call: &call, Text: "Native candidate; not executed"}); err != nil {
		return err
	}
	_, toolErr := s.execute(ctx, v, approve, s.emit)
	return errors.Join(fatalBatchError(ctx, toolErr), ctx.Err())
}
