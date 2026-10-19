// Purpose: Validate persisted conversation messages and tool result references.
// Pattern: Simple Module.
// Usage: History adapters call Message.Check at decoding and write boundaries.
// Related: internal/agent/model/model.go, internal/agent/tool/call.go.
package model

import (
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (message Message) Check() error {
	if message.Role != "user" && message.Role != "assistant" && message.Role != "tool" && message.Role != "system" {
		return fmt.Errorf("invalid message role")
	}
	if message.Incomplete && (message.Role != "assistant" || len(message.Calls) != 0 || message.ToolCallID != "" || message.ToolName != "") {
		return fmt.Errorf("unfinished drafts must be assistant messages without native tool calls or results")
	}
	size := len(message.Content) + len(message.Reasoning)
	for _, call := range message.Calls {
		size += len(call.Arguments) + len(call.ID) + len(call.Name)
	}
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
