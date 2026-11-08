// Purpose: Decode Claude message/content SSE events without exposing unfinished tool calls.
// Pattern: Response-owned Messages stream state.
// Usage: The protocol reader gives each Claude response one Decoder and native completion state.
// Related: internal/provider/protocol/claude/types.go, internal/provider/protocol/native/state.go.
package claude

import (
	"encoding/json"

	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

type blockDraft struct {
	block                     Block
	text, thinking, signature strings.Builder
	stopped                   bool
}
type Decoder struct {
	blocks []*blockDraft
	usage  usage
	reason string
	bytes  int
}

func (d *Decoder) Consume(content []byte, single bool, state *native.State, emit func(modelapi.Event) error) error {
	if len(content) > modelapi.MaxResponseBytes {
		return &modelapi.OutputLimitError{Reason: "Claude response frame byte limit"}
	}
	var frame struct {
		Type  string          `json:"type"`
		Index int             `json:"index"`
		Error json.RawMessage `json:"error"`
		Block json.RawMessage `json:"content_block"`
		Delta struct {
			Type, Text, Thinking, Signature string
			Partial                         string `json:"partial_json"`
			Reason                          string `json:"stop_reason"`
		} `json:"delta"`
		Usage   *usage `json:"usage"`
		Message struct {
			Usage *usage `json:"usage"`
		} `json:"message"`
	}
	if err := json.Unmarshal(content, &frame); err != nil {
		return faults.Errorf(diagcatalog.ClaudeInvalidClaudeEvent, err)
	}
	if single {
		return d.single(content, state, emit)
	}
	if frame.Type == "error" {
		return state.EmitFailedUsage(transport.ParseError(0, content, ""), emit)
	}
	switch frame.Type {
	case "message_start":
		state.Received = true
		return d.setUsage(frame.Message.Usage, state)
	case "content_block_start":
		if frame.Index != len(d.blocks) || frame.Index >= 1024 {
			return faults.Errorf(diagcatalog.ClaudeInvalidClaudeBlockIndex)
		}
		if err := d.addBytes(len(frame.Block)); err != nil {
			return err
		}
		var b Block
		if err := json.Unmarshal(frame.Block, &b); err != nil {
			return err
		}
		switch b.Type {
		case "text", "thinking", "redacted_thinking", "tool_use":
		default:
			return faults.Errorf(diagcatalog.ClaudeUnsupportedClaudeContentType)
		}
		draft := &blockDraft{block: b}
		draft.text.WriteString(b.Text)
		draft.thinking.WriteString(b.Thinking)
		draft.signature.WriteString(b.Signature)
		d.blocks = append(d.blocks, draft)
		if b.Type == "tool_use" {
			call, err := state.Calls.At(frame.Index)
			if err != nil {
				return err
			}
			call.ID, call.Name = b.ID, b.Name
		}
		if b.Text != "" || b.Thinking != "" {
			return emit(modelapi.Event{Delta: b.Text, Thinking: b.Thinking, Streaming: state.Streaming})
		}
	case "content_block_delta":
		if frame.Index < 0 || frame.Index >= len(d.blocks) || d.blocks[frame.Index].stopped {
			return faults.Errorf(diagcatalog.ClaudeClaudeDeltaHasNoActiveBlock)
		}
		b := d.blocks[frame.Index]
		if err := d.addBytes(len(frame.Delta.Text) + len(frame.Delta.Thinking) + len(frame.Delta.Signature) + len(frame.Delta.Partial)); err != nil {
			return err
		}
		switch frame.Delta.Type {
		case "text_delta":
			if b.block.Type != "text" {
				return faults.Errorf(diagcatalog.ClaudeClaudeTextDeltaChangedBlockType)
			}
			b.text.WriteString(frame.Delta.Text)
			state.Usage = nil
			return emit(modelapi.Event{Delta: frame.Delta.Text, Streaming: state.Streaming})
		case "thinking_delta":
			if b.block.Type != "thinking" {
				return faults.Errorf(diagcatalog.ClaudeClaudeThinkingDeltaChangedBlockType)
			}
			b.thinking.WriteString(frame.Delta.Thinking)
			state.Usage = nil
			return emit(modelapi.Event{Thinking: frame.Delta.Thinking, Streaming: state.Streaming})
		case "signature_delta":
			if b.block.Type != "thinking" {
				return faults.Errorf(diagcatalog.ClaudeClaudeSignatureHasNoThinkingBlock)
			}
			b.signature.WriteString(frame.Delta.Signature)
			if frame.Delta.Signature != "" {
				return emit(modelapi.Event{Progress: true})
			}
		case "input_json_delta":
			if b.block.Type != "tool_use" {
				return faults.Errorf(diagcatalog.ClaudeClaudeArgumentsHaveNoToolBlock)
			}
			call, err := state.Calls.At(frame.Index)
			if err != nil {
				return err
			}
			if err := call.Append(frame.Delta.Partial); err != nil {
				return err
			}
			if frame.Delta.Partial != "" {
				return emit(modelapi.Event{Progress: true})
			}
			return nil
		default:
			return faults.Errorf(diagcatalog.ClaudeUnsupportedClaudeContentDelta)
		}
	case "content_block_stop":
		if frame.Index < 0 || frame.Index >= len(d.blocks) || d.blocks[frame.Index].stopped {
			return faults.Errorf(diagcatalog.ClaudeInvalidClaudeBlockStop)
		}
		b := d.blocks[frame.Index]
		b.stopped = true
		b.block.Text, b.block.Thinking, b.block.Signature = b.text.String(), b.thinking.String(), b.signature.String()
		if b.block.Type == "tool_use" {
			call, err := state.Calls.At(frame.Index)
			if err != nil {
				return err
			}
			if call.Arguments.Len() == 0 {
				if err := call.Append(string(b.block.Input)); err != nil {
					return err
				}
			}
			b.block.Input = json.RawMessage(call.Arguments.String())
		}
	case "message_delta":
		d.reason = frame.Delta.Reason
		if err := d.setUsage(frame.Usage, state); err != nil {
			return err
		}
	case "message_stop":
		return d.finish(state, emit)
	}
	return nil
}

func (d *Decoder) addBytes(n int) error {
	if n > modelapi.MaxResponseBytes-d.bytes {
		return &modelapi.OutputLimitError{Reason: "Claude response byte limit"}
	}
	d.bytes += n
	return nil
}
