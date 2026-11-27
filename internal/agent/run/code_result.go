// Purpose: Deliver committed values to code and retain inputs that need model inspection.
// Pattern: Result projection at the host boundary.
// Usage: acceptResult records effects before making these values available to the script.
// Related: internal/agent/run/action_result.go, internal/agent/run/codemode.go, internal/agent/run/code_fallback.go.
package run

import (
	"crypto/rand"
	"encoding/json"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/redaction"
	"github.com/keonho-kim/meca/internal/tools/codemode"
)

func (s *execution) acceptCodeResult(v validatedCall, outcome toolapi.Result, cause error) error {
	content := outcome.Content
	if len(content) == 0 {
		content = json.RawMessage(`null`)
	}
	content, err := redaction.RedactJSON(content, s.runner.Redact)
	if err != nil {
		return err
	}
	state := s.code
	if state != nil {
		if toolapi.IsPagedObservation(v.call.Name) {
			state.transient = true
		} else {
			state.stable = true
		}
		reply := codemode.Reply{Value: content}
		if outcome.Failure != nil {
			reply.Error = s.runner.Redact(outcome.Failure.Error())
		}
		if cause != nil {
			reply.Error = s.runner.Redact(cause.Error())
		}
		reply.Error = excerpt(reply.Error, codemode.MaxErrorBytes)
		state.replies[v.call.ExecutionKey()] = reply
		if reply.Error != "" && (state.lastFailure == nil || outcome.Failure == nil || outcome.Failure.Code != diag.BatchNotExecuted) {
			failureContent := content
			if len(failureContent) > 64<<10 {
				failureContent, _ = json.Marshal(map[string]any{"execution_id": v.call.ExecutionID, "excerpt": excerpt(string(content), 64<<10)})
			}
			failure := outcome.Failure
			if failure != nil {
				copy := *failure
				copy.Detail = excerpt(copy.Detail, codemode.MaxErrorBytes)
				copy.Next = excerpt(copy.Next, codemode.MaxErrorBytes)
				failure = &copy
			}
			body, err := json.Marshal(struct {
				Failure *faults.Failure `json:"failure"`
				Output  json.RawMessage `json:"output"`
			}{failure, failureContent})
			if err != nil {
				return err
			}
			state.lastFailure = &codeObservation{Tool: v.call.Name, Value: body, Source: outcome.Source}
		}
		if errors.Is(cause, errApprovalDenied) {
			state.terminal = cause
		}
		code := faults.Code(cause)
		if outcome.Failure != nil && diag.Priority(outcome.Failure.Code) >= diag.Priority(code) {
			code = outcome.Failure.Code
		}
		if diag.Semantic(code) == diag.PermissionDenied || diag.Semantic(code) == diag.IntegrityFailed || diag.Semantic(code) == diag.PersistenceFailed {
			state.terminal = cause
			if state.terminal == nil {
				state.terminal = outcome.Failure
			}
		}
		if outcome.Failure != nil && outcome.Failure.Code == diag.InstructionChanged {
			state.boundary = true
		}
	}
	if cause != nil || outcome.Failed {
		return nil
	}
	if outcome.Skill != nil || outcome.SkillFile != nil {
		messageID := ""
		if state != nil {
			messageID = state.messageID
		} else {
			messageID = "skill_" + rand.Text()
		}
		message := modelapi.Message{ID: messageID, Role: "tool", Content: string(content), ToolCallID: v.call.ID, ToolName: v.call.Name, Source: outcome.Source}
		if outcome.Skill != nil {
			if err := s.retainSkillRead(*outcome.Skill, message); err != nil {
				return err
			}
		}
		if outcome.SkillFile != nil {
			if err := s.retainSkillReference(*outcome.SkillFile, message); err != nil {
				return err
			}
		}
	}
	if state != nil && (outcome.Skill != nil || outcome.SkillFile != nil || len(outcome.Images) != 0 || outcome.Depart) {
		state.boundary = true
		state.observations = append(state.observations, codeObservation{Tool: v.call.Name, Value: content, Source: outcome.Source})
		state.images = append(state.images, outcome.Images...)
	} else if state != nil {
		return state.fallback.retain(codeObservation{Tool: v.call.Name, Value: content, Source: outcome.Source})
	}
	return nil
}
