// Purpose: Parse parent request DTOs into cache-free native dialect inputs without copying opaque bodies.
// Pattern: Concrete request-value adapter.
// Usage: Native full and cached request paths share this conversion before dialect encoding.
// Related: internal/provider/protocol/input.go, internal/provider/protocol/native/input.go.
package protocol

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

func nativeMessage(message Message) (native.Input, error) {
	calls, err := message.functionCalls()
	if err != nil {
		return native.Input{}, err
	}
	value := native.Input{Role: message.Role, ToolCallID: message.ToolCallID, ToolName: message.ToolName, Calls: calls, State: message.continuation}
	switch content := message.Content.(type) {
	case string:
		value.Text = content
	case []native.ContentPart:
		value.Parts = content
	case nil:
	default:
		return value, fmt.Errorf("native message requires text or image parts")
	}
	return value, nil
}
