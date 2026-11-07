// Purpose: Define the Claude Messages request, signed content and tool wire values.
// Pattern: Typed provider boundary.
// Usage: The parent protocol adapter constructs requests; the Claude decoder retains completed blocks.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/claude/stream.go.
package claude

import "encoding/json"

type Block struct {
	extra     map[string]json.RawMessage
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	Data      string          `json:"data,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	Source    *ImageSource    `json:"source,omitempty"`
}
type ImageSource struct {
	Type  string `json:"type"`
	Media string `json:"media_type"`
	Data  string `json:"data"`
}
type Message struct {
	Role    string  `json:"role"`
	Content []Block `json:"content"`
}
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"input_schema"`
}
type Request struct {
	Model        string    `json:"model"`
	System       string    `json:"system,omitempty"`
	Messages     []Message `json:"messages"`
	Tools        []Tool    `json:"tools,omitempty"`
	MaxTokens    int       `json:"max_tokens"`
	Stream       bool      `json:"stream"`
	Thinking     any       `json:"thinking,omitempty"`
	OutputConfig any       `json:"output_config,omitempty"`
}
