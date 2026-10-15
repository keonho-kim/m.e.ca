// Purpose: Reference an unchanged read only while its original body remains in active context.
// Pattern: Context projection after fresh scoped execution.
// Usage: action stores compact repeated results while retaining raw tool events for auditing.
// Related: internal/agent/run/action.go, internal/agent/run/memory.go.
package run

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) contextResult(call toolapi.Call, result string, success bool) string {
	if !success {
		return result
	}
	read := false
	for _, d := range s.definitions {
		if d.Name == call.Name && d.Effect == policy.Read {
			read = true
		}
	}
	if !read {
		return result
	}
	// A failed operation may need the exact anchors again, even if the bytes did not change.
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Role != "tool" {
			continue
		}
		var feedback struct{ Failure *toolapi.Failure }
		if json.Unmarshal([]byte(s.messages[i].Content), &feedback) == nil && feedback.Failure != nil {
			return result
		}
		break
	}
	for i := len(s.messages) - 1; i >= 0; i-- {
		previous := s.messages[i]
		if previous.Role == "tool" && previous.ToolName == call.Name && previous.Content == result && previous.ToolCallID != "" {
			body, err := json.Marshal(struct {
				Unchanged bool   `json:"unchanged"`
				Source    string `json:"source_call_id"`
			}{true, previous.ToolCallID})
			if err == nil {
				return string(body)
			}
		}
	}
	return result
}
