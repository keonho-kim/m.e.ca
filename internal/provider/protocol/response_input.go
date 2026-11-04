// Purpose: Convert cache-owned conversation DTOs into ordered Responses input values.
// Pattern: Conversation-to-wire adapter.
// Usage: Whole requests and per-message cached encodings use the same conversion.
// Related: internal/provider/protocol/input.go, internal/provider/protocol/responses/request.go.
package protocol

import "encoding/json"

func responseInput(messages []Message) ([]map[string]any, error) {
	var input []map[string]any
	for _, message := range messages {
		if message.Role == "tool" {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": message.ToolCallID, "output": message.Content})
			continue
		}
		if message.Content != "" || len(message.Calls) == 0 {
			input = append(input, map[string]any{"role": message.Role, "content": responseContent(message)})
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
