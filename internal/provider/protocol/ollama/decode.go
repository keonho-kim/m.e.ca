// Purpose: Decode native Ollama chat records and their mandatory completion marker.
// Pattern: Typed NDJSON boundary.
// Usage: The shared stream reader selects Decode for Ollama responses.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/native/frame.go.
package ollama

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

func Decode(content []byte) (native.Frame, error) {
	var wire struct {
		Message native.Message  `json:"message"`
		Done    *bool           `json:"done"`
		Reason  string          `json:"done_reason"`
		Input   *int64          `json:"prompt_eval_count"`
		Output  *int64          `json:"eval_count"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(content, &wire); err != nil {
		return native.Frame{}, fmt.Errorf("invalid model stream: %w", err)
	}
	value := native.Frame{Message: wire.Message, Input: wire.Input, Output: wire.Output, Done: wire.Done, Reason: &wire.Reason}
	if len(wire.Error) > 0 && string(wire.Error) != "null" && string(wire.Error) != `""` {
		return value, transport.ParseError(0, content, "")
	}
	if wire.Done == nil {
		return value, fmt.Errorf("missing Ollama done state")
	}
	return value, nil
}
