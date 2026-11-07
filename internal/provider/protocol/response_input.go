// Purpose: Convert cache-owned conversation DTOs into ordered Responses input values.
// Pattern: Conversation-to-wire adapter.
// Usage: Whole requests and per-message cached encodings use the same conversion.
// Related: internal/provider/protocol/input.go, internal/provider/protocol/responses/request.go.
package protocol

func responseInput(messages []Message) ([]map[string]any, error) {
	var input []map[string]any
	for _, message := range messages {
		if len(message.continuation) > 0 {
			items, err := openAIHistory(message)
			if err != nil {
				return nil, err
			}
			input = append(input, items...)
			continue
		}
		if message.Role == "tool" {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": message.ToolCallID, "output": message.Content})
			continue
		}
		calls, err := message.functionCalls()
		if err != nil {
			return nil, err
		}
		if message.Content != "" || len(calls) == 0 {
			input = append(input, map[string]any{"role": message.Role, "content": responseContent(message)})
		}
		for _, call := range calls {
			input = append(input, map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Name, "arguments": string(call.Arguments)})
		}
	}
	return input, nil
}
