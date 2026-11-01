// Purpose: Preserve completed native history and distinguish unfinished drafts from observed work.
// Pattern: Pure projection.
// Usage: Context assembly converts each persisted message through conversationMessage.
// Related: internal/agent/context/request.go.
package context

import (
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func conversationMessage(message modelapi.Message, paired bool) (modelapi.Message, error) {
	if message.Incomplete {
		if err := message.Check(); err != nil {
			return modelapi.Message{}, err
		}
		body, err := json.Marshal(message.Content)
		if err != nil {
			return modelapi.Message{}, err
		}
		return modelapi.Message{Role: "user", Content: "Unfinished assistant draft (interrupted; not evidence of completed work):\n" + string(body)}, nil
	}
	if message.Role == "system" || !paired && (len(message.Calls) > 0 || message.Role == "tool") {
		// Runtime observations have content, but only assistant replies own
		// reasoning that can be replayed under a provider's message contract.
		if message.Role != "assistant" {
			message.Reasoning = ""
		}
		record := struct {
			Role       string         `json:"role"`
			Content    string         `json:"content,omitempty"`
			Reasoning  string         `json:"reasoning,omitempty"`
			Calls      []toolapi.Call `json:"calls,omitempty"`
			ToolCallID string         `json:"tool_call_id,omitempty"`
			ToolName   string         `json:"tool_name,omitempty"`
		}{message.Role, message.Content, message.Reasoning, message.Calls, message.ToolCallID, message.ToolName}
		body, err := json.Marshal(record)
		if err != nil {
			return modelapi.Message{}, err
		}
		return modelapi.Message{Role: "user", Images: message.Images, Content: "Earlier notes and observations:\n" + string(body)}, nil
	}
	if message.Role != "user" && message.Role != "assistant" && message.Role != "tool" {
		return modelapi.Message{}, fmt.Errorf("invalid conversation role")
	}
	return modelapi.Message{Images: message.Images, Role: message.Role, Content: message.Content, Reasoning: message.Reasoning, ReasoningState: message.ReasoningState, ProviderID: message.ProviderID, ModelID: message.ModelID, Calls: message.Calls, ToolCallID: message.ToolCallID, ToolName: message.ToolName}, nil
}
