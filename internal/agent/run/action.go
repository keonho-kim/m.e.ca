// Purpose: Collect one tool's arguments sequentially, freeze the call and update observed progress.
// Pattern: State Machine (tool dialogue).
// Usage: Invoked once per TOOL or CHECK decision, before the existing approval/execution boundary.
// Related: internal/agent/dialogue/dialogue.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/dialogue"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *task) action(ctx context.Context, decision dialogue.Decision, approve Approver) error {
	var definition *agent.ToolDefinition
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
	if decision.Kind == dialogue.Check && definition.Effect == policy.Change {
		return fmt.Errorf("CHECK cannot mutate files")
	}
	s.checkpoint.Tool = definition.Name
	s.checkpoint.Arguments = map[string]json.RawMessage{}
	if err := s.collectArguments(ctx, *definition); err != nil {
		return err
	}
	content, err := json.Marshal(s.checkpoint.Arguments)
	if err != nil {
		return err
	}
	call := agent.ToolCall{ID: "call_" + rand.Text(), Name: definition.Name, Arguments: content}
	if err = call.Check(); err != nil {
		return err
	}
	signature := sha256.Sum256(append([]byte(call.Name+"\n"), content...))
	mutating := definition.Effect != policy.Read && decision.Kind != dialogue.Check
	if mutating && s.mutations[signature] {
		return fmt.Errorf("repeated identical side-effect proposal stopped; inspect state before retrying")
	}
	s.checkpoint.Phase = "approval"
	s.checkpoint.Parameter = ""
	if err = s.persist(ctx); err != nil {
		return err
	}
	// The compiled call enters the audit transcript only after every argument is valid.
	if err = s.save(ctx, agent.Message{Role: "assistant", Calls: []agent.ToolCall{call}}); err != nil {
		return err
	}
	s.checkpoint.Checking = decision.Kind == dialogue.Check
	result, failed, toolErr := s.execute(ctx, s.planning, call, approve, s.emit)
	s.worked = true
	s.result.HadToolFailure = s.result.HadToolFailure || failed || toolErr != nil
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if toolErr != nil {
		encoded, encodeErr := json.Marshal(map[string]string{"error": s.runner.Redact(toolErr.Error())})
		if encodeErr != nil {
			return encodeErr
		}
		result = string(encoded)
	}
	if err = s.save(ctx, agent.Message{Role: "tool", Content: result, ToolCallID: call.ID, ToolName: call.Name}); err != nil {
		return err
	}
	if errors.Is(toolErr, errApprovalDenied) {
		return toolErr
	}
	s.checkpoint.Phase = "observation"
	s.checkpoint.Arguments = nil
	s.checkpoint.Observation = excerpt(call.Name+" result:\n"+s.runner.Redact(result), 8192)
	if failed || toolErr != nil {
		s.checkpoint.Evidence = ""
		s.checkpoint.NoProgress++
		return s.persist(ctx)
	}
	if mutating {
		s.mutations[signature] = true
		s.changed = true
		s.checkpoint.Evidence = ""
	}
	fingerprint := sha256.Sum256(append(signature[:], []byte(result)...))
	if s.seen[fingerprint] {
		s.checkpoint.NoProgress++
	} else {
		s.seen[fingerprint] = true
		s.checkpoint.NoProgress = 0
	}
	if decision.Kind == dialogue.Check && (!s.changed || definition.Effect == policy.Command) {
		s.checkpoint.Evidence = excerpt(call.Name+" "+string(content)+" => successful result", 4000)
	}
	return s.persist(ctx)
}
