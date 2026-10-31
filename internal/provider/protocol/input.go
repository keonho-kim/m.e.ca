// Purpose: Encode text and image request parts independently of received stream chunks.
// Pattern: Request-only transport DTOs.
// Usage: Messages builds these values; chat and Responses adapters encode them.
// Related: internal/provider/protocol/images.go, internal/provider/protocol/wire.go.
package protocol

import "encoding/json"

// Message is a request DTO. Content is text or []contentPart, selected before
// serialization; received chunk content remains a separate string contract.
type Message struct {
	encoded          []byte
	source           *sourceIdentity
	sourceBytes      int
	hasImages        bool
	ReasoningDetails []json.RawMessage `json:"reasoning_details,omitempty"`
	ToolCallID       string            `json:"tool_call_id,omitempty"`
	ToolName         string            `json:"tool_name,omitempty"`
	Role             string            `json:"role,omitempty"`
	Content          any               `json:"content"`
	Thinking         string            `json:"thinking,omitempty"`
	Reasoning        string            `json:"reasoning,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	Calls            json.RawMessage   `json:"tool_calls,omitempty"`
	Images           []string          `json:"images,omitempty"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}
type imageURL struct {
	URL string `json:"url"`
}

// HasImages distinguishes multimodal content from text-only tokenization inputs.
func (m Message) HasImages() bool {
	parts, _ := m.Content.([]contentPart)
	return m.hasImages || len(parts) > 0 || len(m.Images) > 0
}
