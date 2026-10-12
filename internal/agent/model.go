// Purpose: Define provider-independent model messages and the generation contract.
// Pattern: Consumer-owned I/O contract.
// Usage: Implemented by provider adapters and consumed by the agent loop.
// Related: internal/provider/httpclient/client.go, internal/provider/httpclient/stream.go, internal/tools/local.

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode"
	"unicode/utf8"
)

const (
	MaxArgumentsBytes = 256 * 1024
	MaxToolCalls      = 32
	MaxResponseBytes  = 4 * 1024 * 1024
)

type Message struct {
	ID         string     `json:"id,omitempty"`
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Calls      []ToolCall `json:"calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolName   string     `json:"tool_name,omitempty"`
	ProviderID string     `json:"provider_id,omitempty"`
	ModelID    string     `json:"model_id,omitempty"`
	Thinking   *bool      `json:"thinking,omitempty"`
	Planning   bool       `json:"planning,omitempty"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (call ToolCall) Check() error {
	if call.ID == "" || len(call.ID) > 128 || !utf8.ValidString(call.ID) {
		return fmt.Errorf("invalid Tool Call ID")
	}
	for _, char := range call.ID {
		if unicode.IsControl(char) {
			return fmt.Errorf("invalid Tool Call ID")
		}
	}
	if call.Name == "" || len(call.Name) > 128 {
		return fmt.Errorf("invalid tool name")
	}
	for _, char := range call.Name {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_') {
			return fmt.Errorf("invalid tool name")
		}
	}
	if len(call.Arguments) > MaxArgumentsBytes || !json.Valid(call.Arguments) {
		return fmt.Errorf("invalid or oversized Tool Call arguments")
	}
	var arguments map[string]json.RawMessage
	if err := json.Unmarshal(call.Arguments, &arguments); err != nil || arguments == nil {
		return fmt.Errorf("Tool Call arguments must be a JSON object")
	}
	return nil
}

type ModelRequest struct {
	ModelID   string
	Messages  []Message
	MaxOutput int
	Thinking  *bool
}

type ModelEvent struct {
	Delta        string
	Thinking     string
	Calls        []ToolCall
	Done         bool
	Streaming    bool
	InputTokens  int
	OutputTokens int
}

type Generator interface {
	Generate(context.Context, ModelRequest, func(ModelEvent) error) error
}
