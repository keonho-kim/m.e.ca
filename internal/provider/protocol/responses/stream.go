// Purpose: Translate Responses deltas and reconcile them with completed output.
// Pattern: Response-owned stream adapter.
// Usage: The SSE reader dispatches OpenAI and LM Studio JSON frames here.
// Related: internal/provider/protocol/responses/complete.go, internal/provider/protocol/responses/continuation.go.
package responses

import (
	"encoding/json"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

type Decoder struct {
	text       outputText
	sequences  []sequenceRange
	indices    map[string]int
	identities map[int]string
	items      map[string]int
	Retain     bool
	output     map[int]json.RawMessage
	bytes      int
}

func (s *Decoder) Consume(content []byte, single bool, state *native.State, emit func(modelapi.Event) error) error {
	if single {
		var response response
		if err := json.Unmarshal(content, &response); err != nil {
			return err
		}
		return s.finish(response, "response.completed", state, emit)
	}
	var frame struct {
		Sequence     *int64          `json:"sequence_number"`
		ContentIndex int             `json:"content_index"`
		SummaryIndex int             `json:"summary_index"`
		Part         contentPart     `json:"part"`
		Text         string          `json:"text"`
		Refusal      string          `json:"refusal"`
		Error        json.RawMessage `json:"error"`
		Type         string          `json:"type"`
		Delta        string          `json:"delta"`
		Response     response        `json:"response"`
		Item         json.RawMessage `json:"item"`
		OutputIndex  *int            `json:"output_index"`
		ItemID       string          `json:"item_id"`
		Arguments    string          `json:"arguments"`
	}
	if err := json.Unmarshal(content, &frame); err != nil {
		return faults.Errorf(diagcatalog.ResponsesInvalidResponsesEvent, err)
	}
	if frame.Sequence != nil {
		known, err := s.seenSequence(*frame.Sequence)
		if err != nil || known {
			return err
		}
	}
	index := 0
	if frame.OutputIndex != nil {
		index = *frame.OutputIndex
		if index < 0 || index >= 1024 {
			return faults.Errorf(diagcatalog.ResponsesInvalidResponsesOutputIndex)
		}
		if frame.ItemID != "" {
			if previous, ok := s.items[frame.ItemID]; ok && previous != index {
				return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.ResponsesResponsesFunctionIdentityChanged)}
			}
			if err := s.bindItem(index, frame.ItemID); err != nil {
				return err
			}
		}
	} else if frame.ItemID != "" {
		var known bool
		index, known = s.indices[frame.ItemID]
		if !known {
			index, known = s.items[frame.ItemID]
		}
		if !known {
			return faults.Errorf(diagcatalog.ResponsesInvalidResponsesOutputIndex)
		}
	}
	if len(frame.Error) > 0 && string(frame.Error) != "null" {
		return transport.ParseError(0, content, "")
	}
	if frame.Type == "" {
		return faults.Errorf(diagcatalog.ResponsesMissingResponsesEventType)
	}
	switch frame.Type {
	case "response.output_text.delta", "response.refusal.delta", "response.reasoning_text.delta", "response.reasoning_summary_text.delta":
		kind, part := answerPart, frame.ContentIndex
		if frame.Type == "response.reasoning_text.delta" {
			kind = reasoningPart
		}
		if frame.Type == "response.reasoning_summary_text.delta" {
			kind, part = summaryPart, frame.SummaryIndex
		}
		key := partKey{index, kind, part}
		if p := s.text.items[index][key]; p != nil && p.finalized {
			return nil
		}
		if err := s.text.set(key, frame.Delta, false); err != nil {
			return err
		}
		event := modelapi.Event{Streaming: state.Streaming}
		if frame.Type == "response.output_text.delta" || frame.Type == "response.refusal.delta" {
			event.Delta = frame.Delta
		} else {
			event.Thinking = frame.Delta
		}
		state.Received = true
		state.Usage = nil
		return emit(event)
	case "response.content_part.done", "response.reasoning_summary_part.done":
		kind, part := answerPart, frame.ContentIndex
		if frame.Type == "response.reasoning_summary_part.done" {
			kind, part = summaryPart, frame.SummaryIndex
		} else if frame.Part.Type == "reasoning_text" {
			kind = reasoningPart
		}
		if err := s.text.set(partKey{index, kind, part}, frame.Part.value(), true); err != nil {
			return err
		}
	case "response.output_text.done", "response.refusal.done", "response.reasoning_text.done", "response.reasoning_summary_text.done":
		kind, part, value := answerPart, frame.ContentIndex, frame.Text
		switch frame.Type {
		case "response.refusal.done":
			value = frame.Refusal
		case "response.reasoning_text.done":
			kind = reasoningPart
		case "response.reasoning_summary_text.done":
			kind, part = summaryPart, frame.SummaryIndex
		}
		if err := s.text.set(partKey{index, kind, part}, value, true); err != nil {
			return err
		}
	case "response.output_item.added", "response.output_item.done":
		newContinuation := false
		if frame.Type == "response.output_item.done" && s.Retain {
			if frame.OutputIndex != nil {
				_, known := s.output[*frame.OutputIndex]
				newContinuation = !known
			}
			if err := s.remember(frame.OutputIndex, frame.Item); err != nil {
				return err
			}
		}
		var item outputItem
		if err := json.Unmarshal(frame.Item, &item); err != nil {
			return err
		}
		if item.Type != "function_call" && frame.OutputIndex != nil && item.ID != "" {
			if err := s.bindItem(index, item.ID); err != nil {
				return err
			}
		}
		if item.Type == "function_call" {
			if frame.OutputIndex == nil {
				return faults.Errorf(diagcatalog.ResponsesFunctionOutputHasNoIndex)
			}
			before := state.Calls.Bytes()
			if err := s.call(state, *frame.OutputIndex, item.functionItem, frame.Type == "response.output_item.done"); err != nil {
				return err
			}
			if state.Calls.Bytes() > before {
				return emit(modelapi.Event{Progress: true})
			}
			return nil
		}
		if item.Type != "message" && item.Type != "reasoning" {
			return faults.Errorf(diagcatalog.ResponsesUnsupportedResponsesOutputType)
		}
		if frame.Type == "response.output_item.done" {
			if err := s.text.observeItem(index, item); err != nil {
				return err
			}
		}
		if item.Type == "reasoning" && newContinuation {
			return emit(modelapi.Event{Progress: true})
		}
	case "response.function_call_arguments.delta", "response.function_call_arguments.done":
		index, ok := s.items[frame.ItemID]
		if !ok {
			return faults.Errorf(diagcatalog.ResponsesArgumentsReferToAnUnknownFunctionItem)
		}
		draft, err := state.Calls.At(index)
		if err != nil {
			return err
		}
		if frame.Type == "response.function_call_arguments.delta" {
			if err := draft.Append(frame.Delta); err != nil {
				return err
			}
			if frame.Delta != "" {
				return emit(modelapi.Event{Progress: true})
			}
			return nil
		}
		before := draft.Arguments.Len()
		if err := draft.Final(frame.Arguments); err != nil {
			return err
		}
		if draft.Arguments.Len() > before {
			return emit(modelapi.Event{Progress: true})
		}
		return nil
	case "error":
		return transport.ParseError(0, content, "")
	case "response.completed", "response.failed", "response.incomplete":
		return s.finish(frame.Response, frame.Type, state, emit)
	}
	if !state.Received && s.text.retained > 0 {
		state.Received = true
		return emit(modelapi.Event{Progress: true})
	}
	return nil
}

func (s *Decoder) bindItem(index int, id string) error {
	if len(id) > 256 {
		return faults.Errorf(diagcatalog.ResponsesInvalidResponsesOutputIndex)
	}
	if s.identities == nil {
		s.identities = map[int]string{}
		s.indices = map[string]int{}
	}
	if prev := s.identities[index]; prev != "" && prev != id {
		return faults.Errorf(diagcatalog.ResponsesResponsesFunctionIdentityChanged)
	}
	if prev, ok := s.indices[id]; ok && prev != index {
		return faults.Errorf(diagcatalog.ResponsesInvalidResponsesOutputIndex)
	}
	s.identities[index], s.indices[id] = id, index
	return nil
}
