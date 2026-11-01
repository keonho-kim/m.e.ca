// Purpose: Validate persisted conversation messages and tool result references.
// Pattern: Simple Module.
// Usage: History adapters call Message.Check at decoding and write boundaries.
// Related: internal/agent/model/model.go, internal/agent/tool/call.go.
package model

import (
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (message Message) Check() error {
	if message.Role != "user" && message.Role != "assistant" && message.Role != "tool" && message.Role != "system" {
		return fmt.Errorf("invalid message role")
	}
	if message.Incomplete && (message.Role != "assistant" || len(message.Calls) != 0 || message.ToolCallID != "" || message.ToolName != "") {
		return fmt.Errorf("unfinished drafts must be assistant messages without native tool calls or results")
	}
	if len(message.ReasoningState) > 0 && (message.Role != "assistant" || message.Incomplete) {
		return fmt.Errorf("continuation state requires a completed assistant message")
	}
	if len(message.Images) > 0 && message.Role != "user" && message.Role != "tool" {
		return fmt.Errorf("images require user or tool messages")
	}
	for _, image := range message.Images {
		if err := image.Check(); err != nil {
			return err
		}
	}
	size := message.ContextBytes()
	if size > MaxResponseBytes || len(message.Calls) > toolapi.MaxCalls {
		return fmt.Errorf("message exceeds size limits")
	}
	for _, call := range message.Calls {
		if err := call.Check(); err != nil {
			return err
		}
	}
	if message.Role == "tool" && (message.ToolCallID == "" || len(message.ToolCallID) > 128) {
		return fmt.Errorf("invalid tool result ID")
	}
	return nil
}

// ContextBytes measures retained model content without serializing native state
// or base64 images. Image envelopes reserve the same allowance as model context.
func (message Message) ContextBytes() int {
	size := len(message.Content) + len(message.Reasoning) + len(message.ReasoningState) + len(message.ToolCallID) + len(message.ToolName)
	for _, image := range message.Images {
		size += len(image.Data)*4/3 + 4096
	}
	for _, call := range message.Calls {
		size += len(call.ID) + len(call.Name) + len(call.Arguments)
	}
	return size
}
