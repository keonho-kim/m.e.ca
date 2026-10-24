// Purpose: Encode native function declarations and provider-specific call/result history.
// Pattern: Provider wire adapter.
// Usage: HTTP generation and exact tokenization use the same definitions and messages.
// Related: internal/provider/protocol/request.go, internal/agent/tool/json_schema.go.
package protocol

import (
	"encoding/json"
	"fmt"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

func Tools(definitions []toolapi.Definition) []Tool {
	var tools []Tool
	for _, d := range definitions {
		tools = append(tools, Tool{Type: "function", Function: Function{Name: d.Name, Description: d.Description, Parameters: d.JSONSchema()}})
	}
	return tools
}

func Messages(kind connection.ProviderKind, source []modelapi.Message) ([]Message, error) {
	var messages []Message
	for _, m := range source {
		if m.Role != "system" && m.Role != "user" && m.Role != "assistant" && m.Role != "tool" {
			return nil, fmt.Errorf("invalid message role")
		}
		message := Message{Role: m.Role, Content: m.Content}
		if kind == connection.OpenRouter && m.Role == "assistant" && !m.Incomplete && len(m.ReasoningState) > 0 {
			var err error
			message.ReasoningDetails, err = routerHistory(m.ReasoningState)
			if err != nil {
				return nil, err
			}
		}
		if kind == connection.Ollama {
			message.Thinking = m.Reasoning
		} else if kind == connection.SGLang || kind == connection.LiteLLM {
			message.ReasoningContent = m.Reasoning
		} else {
			message.Reasoning = m.Reasoning
		}
		if len(message.ReasoningDetails) > 0 {
			message.Reasoning = ""
		}
		if m.Role == "tool" {
			if m.ToolCallID == "" || m.ToolName == "" {
				return nil, fmt.Errorf("tool result lacks its call identity")
			}
			if kind == connection.Ollama {
				message.ToolName = m.ToolName
			} else {
				message.ToolCallID = m.ToolCallID
			}
		}
		if len(m.Calls) > 0 {
			if m.Role != "assistant" || len(m.Calls) > toolapi.MaxCalls {
				return nil, fmt.Errorf("invalid assistant tool calls")
			}
			var calls []map[string]any
			for _, call := range m.Calls {
				if err := call.Check(); err != nil {
					return nil, err
				}
				var arguments any = string(call.Arguments)
				if kind == connection.Ollama {
					arguments = call.Arguments
				}
				entry := map[string]any{"type": "function", "function": map[string]any{"name": call.Name, "arguments": arguments}}
				if kind != connection.Ollama {
					entry["id"] = call.ID
				}
				calls = append(calls, entry)
			}
			body, err := json.Marshal(calls)
			if err != nil {
				return nil, err
			}
			message.Calls = body
		}
		messages = append(messages, message)
	}
	return messages, nil
}
