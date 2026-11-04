// Purpose: Express observed wire fields without coupling independent API decoders.
// Pattern: Response-only DTOs.
// Usage: Chat and Ollama parsers return immutable decoded frame values.
// Related: internal/provider/protocol/chat/decode.go, internal/provider/protocol/ollama/decode.go.
package native

import "encoding/json"

type Message struct {
	ReasoningDetails []json.RawMessage `json:"reasoning_details,omitempty"`
	ToolCallID       string            `json:"tool_call_id,omitempty"`
	ToolName         string            `json:"tool_name,omitempty"`
	Role             string            `json:"role,omitempty"`
	Content          string            `json:"content"`
	Thinking         string            `json:"thinking,omitempty"`
	Reasoning        string            `json:"reasoning,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	Calls            json.RawMessage   `json:"tool_calls,omitempty"`
}

type Frame struct {
	Message       Message
	Input, Output *int64
	Cost          *float64
	Reason        *string
	Done          *bool
}
