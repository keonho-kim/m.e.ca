// Purpose: Accept only completed Claude blocks and preserve signed continuation in original order.
// Pattern: Terminal state transition.
// Usage: Claude stream and complete-JSON decoding converge here before native calls are released.
// Related: internal/provider/protocol/claude/stream.go, internal/provider/protocol/claude_request.go.
package claude

import (
	"encoding/json"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

func (d *Decoder) finish(state *native.State, emit func(modelapi.Event) error) error {
	defer func() { *d = Decoder{} }()
	switch d.reason {
	case "end_turn", "stop_sequence", "tool_use":
	case "max_tokens", "model_context_window_exceeded":
		return state.EmitFailedUsage(&modelapi.OutputLimitError{Reason: "Claude output token limit"}, emit)
	default:
		return state.EmitFailedUsage(&transport.PermanentError{Cause: faults.Errorf(diagcatalog.ClaudeClaudeResponseDidNotComplete, d.reason)}, emit)
	}
	blocks := make([]Block, len(d.blocks))
	for i, b := range d.blocks {
		if !b.stopped {
			return faults.Errorf(diagcatalog.ClaudeClaudeResponseEndedWithAnUnfinishedBlock)
		}
		blocks[i] = b.block
	}
	encoded, err := json.Marshal(blocks)
	if err != nil {
		return err
	}
	if len(encoded) > modelapi.MaxResponseBytes {
		return &modelapi.OutputLimitError{Reason: "Claude continuation byte limit"}
	}
	state.Continuation = encoded
	state.Received, state.Finished = true, true
	return nil
}

func (d *Decoder) single(content []byte, state *native.State, emit func(modelapi.Event) error) error {
	var value struct {
		Type    string
		Error   json.RawMessage
		Content []Block
		Usage   *usage
		Reason  string `json:"stop_reason"`
	}
	if err := json.Unmarshal(content, &value); err != nil {
		return err
	}
	if value.Type == "error" {
		return transport.ParseError(0, content, "")
	}
	if value.Type != "message" {
		return faults.Errorf(diagcatalog.ClaudeInvalidClaudeMessage)
	}
	for i, b := range value.Content {
		switch b.Type {
		case "text", "thinking", "redacted_thinking":
			if err := emit(modelapi.Event{Delta: b.Text, Thinking: b.Thinking}); err != nil {
				return err
			}
		case "tool_use":
			call, err := state.Calls.At(i)
			if err != nil {
				return err
			}
			call.ID, call.Name = b.ID, b.Name
			if err := call.Append(string(b.Input)); err != nil {
				return err
			}
		default:
			return faults.Errorf(diagcatalog.ClaudeUnsupportedClaudeContentType)
		}
		d.blocks = append(d.blocks, &blockDraft{block: b, stopped: true})
	}
	if err := d.setUsage(value.Usage, state); err != nil {
		return err
	}
	d.reason = value.Reason
	return d.finish(state, emit)
}
