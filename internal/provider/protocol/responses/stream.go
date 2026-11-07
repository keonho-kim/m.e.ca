// Purpose: Translate Responses deltas without materializing repeated final text or reasoning.
// Pattern: Response-owned stream adapter.
// Usage: The SSE reader dispatches OpenAI and LM Studio JSON frames here.
// Related: internal/provider/protocol/responses/complete.go, internal/provider/protocol/responses/continuation.go.
package responses

import (
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

type Decoder struct {
	items  map[string]int
	Retain bool
	output map[int]json.RawMessage
	bytes  int
}

func (s *Decoder) Consume(content []byte, single bool, state *native.State, emit func(modelapi.Event) error) error {
	if single {
		var response response
		if err := json.Unmarshal(content, &response); err != nil {
			return err
		}
		return s.finish(response, "response.completed", true, state, emit)
	}
	var frame struct {
		Error       json.RawMessage `json:"error"`
		Type        string          `json:"type"`
		Delta       string          `json:"delta"`
		Response    response        `json:"response"`
		Item        json.RawMessage `json:"item"`
		OutputIndex *int            `json:"output_index"`
		ItemID      string          `json:"item_id"`
		Arguments   string          `json:"arguments"`
	}
	if err := json.Unmarshal(content, &frame); err != nil {
		return fmt.Errorf("invalid Responses event: %w", err)
	}
	if len(frame.Error) > 0 && string(frame.Error) != "null" {
		return transport.ParseError(0, content, "")
	}
	if frame.Type == "" {
		return fmt.Errorf("missing Responses event type")
	}
	switch frame.Type {
	case "response.output_text.delta", "response.refusal.delta", "response.reasoning_text.delta", "response.reasoning_summary_text.delta":
		event := modelapi.Event{Streaming: state.Streaming}
		if frame.Type == "response.output_text.delta" || frame.Type == "response.refusal.delta" {
			event.Delta = frame.Delta
		} else {
			event.Thinking = frame.Delta
		}
		state.Received = true
		state.Usage = nil
		return emit(event)
	case "response.output_item.added", "response.output_item.done":
		if frame.Type == "response.output_item.done" && s.Retain {
			if err := s.remember(frame.OutputIndex, frame.Item); err != nil {
				return err
			}
		}
		var item functionItem
		if err := json.Unmarshal(frame.Item, &item); err != nil {
			return err
		}
		if item.Type == "function_call" {
			if frame.OutputIndex == nil {
				return fmt.Errorf("function output has no index")
			}
			return s.call(state, *frame.OutputIndex, item, frame.Type == "response.output_item.done")
		}
		if item.Type != "message" && item.Type != "reasoning" {
			return fmt.Errorf("unsupported Responses output type")
		}
	case "response.function_call_arguments.delta", "response.function_call_arguments.done":
		index, ok := s.items[frame.ItemID]
		if !ok {
			return fmt.Errorf("arguments refer to an unknown function item")
		}
		draft, err := state.Calls.At(index)
		if err != nil {
			return err
		}
		if frame.Type == "response.function_call_arguments.delta" {
			return draft.Append(frame.Delta)
		}
		return draft.Final(frame.Arguments)
	case "error":
		return transport.ParseError(0, content, "")
	case "response.completed", "response.failed", "response.incomplete":
		return s.finish(frame.Response, frame.Type, false, state, emit)
	}
	return nil
}
