// Purpose: Validate one native tool call and orchestrate grounding, approval and execution.
// Pattern: Native tool execution use case.
// Usage: Invoked once per complete native tool call, before the existing approval/execution boundary.
// Related: internal/agent/tool/arguments.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) action(ctx context.Context, reply modelapi.Response, approve Approver) error {
	if reply.Calls[0].Name == "submit_plan" {
		return s.submitPlan(ctx, reply)
	}
	call, reason, intentErr := toolapi.ExtractIntent(reply.Calls[0])
	if intentErr != nil {
		return s.returnToolFeedback(ctx, reply, intentErr.Error())
	}
	var definition *toolapi.Definition
	for i := range s.definitions {
		if s.definitions[i].Name == call.Name {
			definition = &s.definitions[i]
			break
		}
	}
	if err := call.Check(); err != nil {
		return err
	}
	if definition == nil {
		return s.returnToolFeedback(ctx, reply, "Unknown tool; select a declared native function.")
	}
	if err := s.allowTool(definition.Effect); err != nil {
		return s.returnToolFeedback(ctx, reply, err.Error())
	}
	if call.Name == "write_file" || call.Name == "edit_file" {
		s.checkpoint.PendingReview = true
	}
	if len(call.Arguments) > s.limits.MaxArgumentBytes {
		return s.returnToolFeedback(ctx, reply, fmt.Sprintf("Arguments exceed the configured %d-byte limit; narrow the operation. Nothing executed.", s.limits.MaxArgumentBytes))
	}
	arguments, err := toolapi.ValidateArguments(*definition, call.Arguments)
	if err != nil {
		return s.returnToolFeedback(ctx, reply, "Argument validation failed: "+err.Error()+". Correct the arguments; no operation was executed.")
	}
	s.checkpoint.Tool = definition.Name
	s.checkpoint.Arguments = arguments
	s.checkpoint.Checking = false
	if value, ok := arguments["purpose"]; ok {
		var purpose string
		if err := json.Unmarshal(value, &purpose); err != nil {
			return err
		}
		s.checkpoint.Checking = purpose == "check" && definition.Effect == policy.Command
	}
	content := call.Arguments
	if err := s.ground(ctx, call, reason); err != nil {
		return err
	}
	signature := sha256.Sum256(append([]byte(call.Name+"\n"), content...))
	mutating := definition.Effect != policy.Read && !s.checkpoint.Checking
	if mutating && s.mutations[signature] {
		return s.returnToolFeedback(ctx, reply, "Repeated identical side-effect proposal blocked. Inspect current state and narrow or change the operation; previous execution must not be replayed.")
	}
	s.checkpoint.Phase = "approval"
	if err = s.persist(ctx); err != nil {
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
	if err := s.record(ctx, Event{Kind: "tool_proposed", CallID: call.ID, Call: &call, Text: "Grounded native candidate; not executed"}); err != nil {
		return err
	}
	outcome, toolErr := s.execute(ctx, s.planning, call, approve, s.emit)
	result, failed := string(outcome.Content), outcome.Failed
	if errors.Is(toolErr, errRuntimeRecord) {
		return toolErr
	}
	s.result.HadToolFailure = s.result.HadToolFailure || failed || toolErr != nil
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if toolErr != nil || failed {
		failure := outcome.Failure
		if failure == nil {
			failure = toolapi.FailureOf(toolErr)
		}
		if failure == nil {
			failure = &toolapi.Failure{Code: toolapi.ExecutionFailed, Detail: "tool reported failure", Next: "Inspect its output before choosing another action."}
		}
		failure.Detail = s.runner.Redact(failure.Detail)
		encoded, encodeErr := json.Marshal(struct {
			Failure *toolapi.Failure
			Output  json.RawMessage
		}{failure, outcome.Content})
		if encodeErr != nil {
			return encodeErr
		}
		result = string(encoded)
	}

	if err = s.save(ctx, modelapi.Message{Role: "tool", Content: s.contextResult(call, result, !failed && toolErr == nil), ToolCallID: call.ID, ToolName: call.Name}); err != nil {
		return err
	}
	if errors.Is(toolErr, errApprovalDenied) {
		return toolErr
	}
	return s.observeOutcome(ctx, call, content, signature, mutating, definition.Effect, result, outcome, toolErr)
}
