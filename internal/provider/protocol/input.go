// Purpose: Define request DTOs with typed native calls and text/image wire parts.
// Pattern: Request-only transport DTOs.
// Usage: Messages borrows immutable engine calls; concrete request adapters consume these values.
// Related: internal/provider/protocol/tools.go, internal/provider/protocol/native/history.go.
package protocol

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

// Message is a request DTO. Content is text or []native.ContentPart, selected before
// serialization; received chunk content remains a separate string contract.
type Message struct {
	cloudSource      *cloudSourceID
	encoded          []byte
	source           *sourceIdentity
	sourceBytes      int
	hasImages        bool
	continuation     json.RawMessage
	nativeCalls      []toolapi.Call
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

// Native adapters borrow validated engine calls instead of encoding and then
// decoding the chat tool_calls dialect. Raw DTOs still parse at this boundary.
func (m Message) functionCalls() ([]toolapi.Call, error) {
	if m.nativeCalls != nil {
		return m.nativeCalls, nil
	}
	return native.History(m.Calls)
}

// HasImages distinguishes multimodal content from text-only tokenization inputs.
func (m Message) HasImages() bool {
	parts, _ := m.Content.([]native.ContentPart)
	return m.hasImages || len(parts) > 0 || len(m.Images) > 0
}
