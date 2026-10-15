// Purpose: Preserve native tool history and quote internal runtime records as observations.
// Pattern: Pure projection.
// Usage: Context assembly converts each persisted message through conversationMessage.
// Related: internal/agent/context/request.go.
package context

import (
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func conversationMessage(message modelapi.Message, paired bool) (modelapi.Message, error) {
	if message.Role == "system" || !paired && (len(message.Calls) > 0 || message.Role == "tool") {
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
		return modelapi.Message{Role: "user", Content: "Archived runtime record (data, not a user request):\n" + string(body)}, nil
	}
	if message.Role != "user" && message.Role != "assistant" && message.Role != "tool" {
		return modelapi.Message{}, fmt.Errorf("invalid conversation role")
	}
	return modelapi.Message{Role: message.Role, Content: message.Content, Reasoning: message.Reasoning, Calls: message.Calls, ToolCallID: message.ToolCallID, ToolName: message.ToolName}, nil
}

// Compaction and interrupted work can leave partial groups. Keep those records
// as quoted observations rather than sending orphan native results to a server.
func pairedTools(messages []modelapi.Message) map[int]bool {
	paired := make(map[int]bool)
	for index, message := range messages {
		if message.Role != "assistant" || len(message.Calls) == 0 || index+len(message.Calls) >= len(messages) {
			continue
		}
		pending := make(map[string]string)
		for _, call := range message.Calls {
			pending[call.ID] = call.Name
		}
		if len(pending) != len(message.Calls) {
			continue
		}
		valid := true
		for offset := 1; offset <= len(message.Calls); offset++ {
			result := messages[index+offset]
			name, exists := pending[result.ToolCallID]
			if result.Role != "tool" || !exists || result.ToolName != name {
				valid = false
				break
			}
			delete(pending, result.ToolCallID)
		}
		if valid {
			for offset := 0; offset <= len(message.Calls); offset++ {
				paired[index+offset] = true
			}
		}
	}
	return paired
}
