// Purpose: Finalize Responses usage, complete function calls and original continuation exactly once.
// Pattern: Terminal response transition with explicit draft release.
// Usage: Complete JSON and streamed terminal events converge here.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/responses/calls.go.
package responses

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

type response struct {
	Status     string          `json:"status"`
	Error      json.RawMessage `json:"error"`
	Incomplete struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Usage *struct {
		Input  *int64 `json:"input_tokens"`
		Output *int64 `json:"output_tokens"`
	} `json:"usage"`
	Output json.RawMessage `json:"output"`
}

func (s *Decoder) finish(response response, kind string, state *native.State, emit func(modelapi.Event) error) error {
	defer func() { *s = Decoder{Retain: s.Retain} }()
	if response.Usage != nil {
		input, output := response.Usage.Input, response.Usage.Output
		if input != nil && (*input < 0 || *input > modelapi.MaxCallTokens) || output != nil && (*output < 0 || *output > modelapi.MaxCallTokens) {
			return faults.Errorf(diagcatalog.ProtocolInvalidProviderTokenUsage)
		}
		if input != nil && output != nil {
			state.Usage = &modelapi.TokenUsage{Input: *input, Output: *output}
		}
	}
	state.Received = true
	if response.Status != "completed" || kind != "response.completed" {
		detail := "response did not complete"
		if response.Incomplete.Reason != "" {
			detail = response.Incomplete.Reason
		}
		var cause error
		if len(response.Error) > 0 && string(response.Error) != "null" {
			cause = transport.ParseError(0, append(append([]byte(`{"error":`), response.Error...), '}'), "")
			detail = cause.Error()
		}
		if response.Incomplete.Reason == "max_output_tokens" {
			// A truncated response can deliver its only text or partial tool in the terminal frame.
			limit := &modelapi.OutputLimitError{Reason: detail, TokenLimit: true}
			if err := s.finishOutput(response.Output, state, true); err != nil {
				// Unparseable terminal output cannot establish a safe text continuation.
				limit.TokenLimit = false
				return state.EmitFailedUsage(errors.Join(limit, err), emit)
			}
			if err := s.EmitPartial(state.Streaming, emit); err != nil {
				return err
			}
			return state.EmitFailedUsage(limit, emit)
		}
		if cause != nil {
			return state.EmitFailedUsage(faults.Errorf(diagcatalog.ResponsesResponses, cause), emit)
		}
		return state.EmitFailedUsage(&transport.PermanentError{Cause: faults.Errorf(diagcatalog.ResponsesResponses2, detail)}, emit)
	}
	if err := s.finishOutput(response.Output, state, false); err != nil {
		return state.EmitFailedUsage(err, emit)
	}
	if err := s.retain(response.Output, state); err != nil {
		return err
	}
	state.Completion = s.text.completion()
	state.Finished = true
	return nil
}

func (s *Decoder) finishOutput(raw json.RawMessage, state *native.State, partial bool) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	// bytes.Buffer lets jsontext borrow each item's wire bytes until the next read.
	reader := jsontext.NewDecoder(bytes.NewBuffer(raw))
	open, err := reader.ReadToken()
	if err != nil {
		return err
	}
	if open.Kind() != '[' {
		return faults.Errorf(diagcatalog.ResponsesUnsupportedResponsesOutputType)
	}
	count, confirmedCalls := 0, 0
	for reader.PeekKind() != ']' {
		if count >= 1024 {
			return faults.Errorf(diagcatalog.ResponsesInvalidResponsesOutputIndex)
		}
		wire, err := reader.ReadValue()
		if err != nil {
			return err
		}
		if state.Calls.HasIndex(count) || !bytes.Equal(s.output[count], wire) {
			call, err := s.finishItem(count, wire, state, partial)
			if err != nil {
				return err
			}
			if call {
				confirmedCalls++
			}
		}
		count++
	}
	if _, err := reader.ReadToken(); err != nil {
		return err
	}
	if !partial {
		for index := range s.text.items {
			if index >= count {
				s.text.removeParts(index, func(k partKey) bool { return k.kind == answerPart })
			}
		}
		if state.Calls.Count() != confirmedCalls {
			return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.ResponsesIncompleteResponsesFunctionIdentity)}
		}
	}
	return nil
}

func (s *Decoder) finishItem(index int, wire []byte, state *native.State, partial bool) (bool, error) {
	var item outputItem
	if err := json.Unmarshal(wire, &item); err != nil {
		return false, err
	}
	if item.ID != "" {
		if err := s.bindItem(index, item.ID); err != nil {
			return false, err
		}
	}
	// A terminal message owns its body. A reasoning item may include only opaque state.
	if !partial && item.Type != "message" {
		s.text.removeParts(index, func(k partKey) bool { return k.kind == answerPart })
	}
	switch item.Type {
	case "message", "reasoning":
		if partial && len(item.Content) == 0 && len(item.Summary) == 0 {
			return false, nil
		}
		return false, s.text.observeItem(index, item)
	case "function_call":
		if partial {
			_, err := state.Calls.At(index)
			return false, err
		}
		return true, s.call(state, index, item.functionItem, true)
	default:
		return false, faults.Errorf(diagcatalog.ResponsesUnsupportedResponsesOutputType)
	}
}
