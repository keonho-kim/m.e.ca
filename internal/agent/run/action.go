// Purpose: Orchestrate PFC, self-grounding and observed progress for one tool decision.
// Pattern: State Machine (PFC execution).
// Usage: Invoked once per JSON tool entry, before the existing approval/execution boundary.
// Related: internal/agent/pfc/pfc.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/pfc"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *task) action(ctx context.Context, decision pfc.Decision, approve Approver) error {
	var definition *toolapi.Definition
	for i := range s.definitions {
		if s.definitions[i].Name == decision.Name {
			definition = &s.definitions[i]
			break
		}
	}
	if definition == nil {
		s.checkpoint.Observation = "Unknown tool: " + decision.Name
		s.checkpoint.NoProgress++
		return nil
	}
	if _, err := policy.Decide(s.runner.Settings, s.planning, definition.Effect); err != nil {
		return err
	}
	s.checkpoint.Tool = definition.Name
	s.checkpoint.Arguments = map[string]json.RawMessage{}
	if err := s.collectArguments(ctx, *definition); err != nil {
		var failure *toolapi.Failure
		if !errors.As(err, &failure) {
			return err
		}
		s.checkpoint.Observation = failure.Error()
		s.checkpoint.NoProgress++
		return s.persist(ctx)
	}
	content, err := json.Marshal(s.checkpoint.Arguments)
	if err != nil {
		return err
	}
	call := toolapi.Call{ID: "call_" + rand.Text(), Name: definition.Name, Arguments: content}
	if err = call.Check(); err != nil {
		return err
	}
	if err := s.ground(ctx, call); err != nil {
		return err
	}
	signature := sha256.Sum256(append([]byte(call.Name+"\n"), content...))
	mutating := definition.Effect != policy.Read && !s.checkpoint.Checking
	if mutating && s.mutations[signature] {
		s.checkpoint.Observation = "Repeated identical side-effect proposal blocked. Inspect current state and narrow or change the operation; previous execution must not be replayed."
		s.checkpoint.NoProgress++
		return s.persist(ctx)
	}
	s.checkpoint.Phase = "approval"
	s.checkpoint.Parameter = ""
	if err = s.persist(ctx); err != nil {
		return err
	}
	// The compiled call enters the audit transcript only after every argument is valid.
	if err = s.save(ctx, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{call}}); err != nil {
		return err
	}
	if err := s.record(ctx, Event{Kind: "tool_proposed", CallID: call.ID, Call: &call, Text: "Grounded PFC candidate; not executed"}); err != nil {
		return err
	}
	outcome, toolErr := s.execute(ctx, s.planning, call, approve, s.emit)
	result, failed := string(outcome.Content), outcome.Failed
	if errors.Is(toolErr, errRuntimeRecord) {
		return toolErr
	}
	s.worked = true
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

	if err = s.save(ctx, modelapi.Message{Role: "tool", Content: result, ToolCallID: call.ID, ToolName: call.Name}); err != nil {
		return err
	}
	if errors.Is(toolErr, errApprovalDenied) {
		return toolErr
	}
	return s.observeOutcome(ctx, call, content, signature, mutating, definition.Effect, result, outcome, toolErr)
}
