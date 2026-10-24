// Purpose: Encode stateless LM Studio Responses requests with full ordered role history.
// Pattern: Provider wire adapter.
// Usage: Generate sends this payload to /v1/responses and uses the shared retry lifecycle.
// Related: internal/provider/protocol/lmstudio_stream.go, internal/provider/protocol/request.go.
package protocol

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func lmPayload(model string, messages []Message, maxOutput int, effort string, temperature *float64, definitions []toolapi.Definition) ([]byte, error) {
	input, err := lmMessages(messages)
	if err != nil {
		return nil, err
	}
	body := struct {
		Tools       []map[string]any  `json:"tools,omitempty"`
		ToolChoice  string            `json:"tool_choice,omitempty"`
		Parallel    *bool             `json:"parallel_tool_calls,omitempty"`
		Temperature *float64          `json:"temperature,omitempty"`
		Model       string            `json:"model"`
		Input       []map[string]any  `json:"input"`
		Stream      bool              `json:"stream"`
		Store       bool              `json:"store"`
		Truncation  string            `json:"truncation"`
		MaxOutput   int               `json:"max_output_tokens"`
		Reasoning   map[string]string `json:"reasoning,omitempty"`
	}{Temperature: temperature, Model: model, Input: input, Stream: true, Store: false, Truncation: "disabled", MaxOutput: maxOutput}
	for _, tool := range Tools(definitions) {
		body.Tools = append(body.Tools, map[string]any{"type": "function", "name": tool.Function.Name, "description": tool.Function.Description, "parameters": tool.Function.Parameters})
	}
	if len(body.Tools) > 0 {
		parallel := true
		body.Parallel, body.ToolChoice = &parallel, "auto"
	}
	if effort != "" {
		body.Reasoning = map[string]string{"effort": effort}
	}
	return json.Marshal(body)
}

func lmMessages(messages []Message) ([]map[string]any, error) {
	var input []map[string]any
	for _, message := range messages {
		if message.Role == "tool" {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": message.ToolCallID, "output": message.Content})
			continue
		}
		if message.Content != "" || len(message.Calls) == 0 {
			input = append(input, map[string]any{"role": message.Role, "content": message.Content})
		}
		if len(message.Calls) == 0 {
			continue
		}
		var calls []struct {
			ID       string
			Function struct{ Name, Arguments string }
		}
		if err := json.Unmarshal(message.Calls, &calls); err != nil {
			return nil, err
		}
		for _, call := range calls {
			input = append(input, map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Function.Name, "arguments": call.Function.Arguments})
		}
	}
	return input, nil
}
