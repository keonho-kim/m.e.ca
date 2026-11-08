// Purpose: Decode Chat Completions frames independently of Ollama and Responses.
// Pattern: Typed wire boundary.
// Usage: The shared stream reader selects Decode for Chat-compatible providers.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/native/frame.go.
package chat

import (
	"encoding/json"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

func Decode(content []byte, single bool) (native.Frame, error) {
	var wire struct {
		Error   json.RawMessage `json:"error"`
		Choices []struct {
			Delta   native.Message `json:"delta"`
			Message native.Message `json:"message"`
			Reason  *string        `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			Input  *int64   `json:"prompt_tokens"`
			Output *int64   `json:"completion_tokens"`
			Cost   *float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(content, &wire); err != nil {
		return native.Frame{}, faults.Errorf(diagcatalog.ChatInvalidModelStream, err)
	}
	var value native.Frame
	if wire.Usage != nil {
		value.Input, value.Output, value.Cost = wire.Usage.Input, wire.Usage.Output, wire.Usage.Cost
	}
	if len(wire.Error) > 0 && string(wire.Error) != "null" && string(wire.Error) != `""` {
		return value, transport.ParseError(0, content, "")
	}
	if len(wire.Choices) > 1 {
		return value, faults.Errorf(diagcatalog.ChatMultipleModelChoicesAreNotSupported)
	}
	if len(wire.Choices) == 0 {
		if wire.Usage == nil {
			return value, faults.Errorf(diagcatalog.ChatMissingModelChoices)
		}
		return value, nil
	}
	choice := wire.Choices[0]
	value.Message, value.Reason = choice.Delta, choice.Reason
	if single {
		value.Message = choice.Message
		if choice.Reason != nil {
			done := true
			value.Done = &done
		}
	}
	return value, nil
}
