// Purpose: Finalize Responses usage, complete function calls and original continuation exactly once.
// Pattern: Terminal response transition with explicit draft release.
// Usage: Complete JSON and streamed terminal events converge here.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/responses/calls.go.
package responses

import (
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
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

func (s *Decoder) finish(response response, kind string, single bool, state *native.State, emit func(modelapi.Event) error) error {
	defer func() { s.items, s.output, s.bytes = nil, nil, 0 }()
	if response.Usage != nil {
		input, output := response.Usage.Input, response.Usage.Output
		if input != nil && (*input < 0 || *input > modelapi.MaxCallTokens) || output != nil && (*output < 0 || *output > modelapi.MaxCallTokens) {
			return fmt.Errorf("invalid provider token usage")
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
			return state.EmitFailedUsage(&modelapi.OutputLimitError{Reason: detail}, emit)
		}
		if cause != nil {
			return state.EmitFailedUsage(fmt.Errorf("responses: %w", cause), emit)
		}
		return state.EmitFailedUsage(&transport.PermanentError{Cause: fmt.Errorf("responses: %s", detail)}, emit)
	}
	if err := s.finishOutput(response.Output, single, state, emit); err != nil {
		return state.EmitFailedUsage(err, emit)
	}
	if err := s.retain(response.Output, state); err != nil {
		return err
	}
	state.Finished = true
	return nil
}

func (s *Decoder) finishOutput(raw json.RawMessage, single bool, state *native.State, emit func(modelapi.Event) error) error {
	if len(raw) == 0 {
		return nil
	}
	// Streaming text has already been emitted. Decode only function identities
	// there, leaving large message/summary/encrypted fields in the raw array.
	var items []functionItem
	var complete []outputItem
	if single {
		if err := json.Unmarshal(raw, &complete); err != nil {
			return err
		}
		for _, item := range complete {
			items = append(items, item.functionItem)
		}
	} else if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	for index, item := range items {
		if item.Type == "function_call" {
			if err := s.call(state, index, item, true); err != nil {
				return err
			}
			continue
		}
		if item.Type != "message" && item.Type != "reasoning" {
			return fmt.Errorf("unsupported Responses output type")
		}
		if !single {
			continue
		}
		for _, part := range complete[index].Content {
			event := modelapi.Event{Delta: part.Text + part.Refusal}
			if item.Type == "reasoning" {
				event.Delta, event.Thinking = "", part.Text
			}
			if err := emit(event); err != nil {
				return err
			}
		}
		for _, summary := range complete[index].Summary {
			if err := emit(modelapi.Event{Thinking: summary.Text}); err != nil {
				return err
			}
		}
	}
	return nil
}
