// Purpose: Translate LM Studio Responses events, terminal failures and actual token usage.
// Pattern: Stateful stream adapter.
// Usage: The SSE reader dispatches LM Studio JSON frames here.
// Related: internal/provider/protocol/lmstudio.go, internal/provider/protocol/stream.go.
package protocol

import (
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/transport"
)

type lmResponse struct {
	Status     string          `json:"status"`
	Error      json.RawMessage `json:"error"`
	Incomplete struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Usage *struct {
		Input  *int64 `json:"input_tokens"`
		Output *int64 `json:"output_tokens"`
	} `json:"usage"`
	Output []lmItem `json:"output"`
}

func (s *streamState) consumeLM(content []byte, single bool, emit func(modelapi.Event) error) error {
	var frame struct {
		Error       json.RawMessage `json:"error"`
		Type        string          `json:"type"`
		Delta       string          `json:"delta"`
		Response    lmResponse      `json:"response"`
		Item        lmItem          `json:"item"`
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
	if !single && frame.Type == "" {
		return fmt.Errorf("missing Responses event type")
	}
	if single {
		if err := json.Unmarshal(content, &frame.Response); err != nil {
			return err
		}
		for index, item := range frame.Response.Output {
			if item.Type == "function_call" {
				if frame.Response.Status != "completed" {
					continue
				}
				if err := s.lmCall(index, item, true); err != nil {
					return err
				}
				continue
			}
			if item.Type != "message" && item.Type != "reasoning" {
				return fmt.Errorf("unsupported Responses output type")
			}
			for _, part := range item.Content {
				event := modelapi.Event{}
				if item.Type == "reasoning" {
					event.Thinking = part.Text
				} else {
					event.Delta = part.Text
				}
				if err := emit(event); err != nil {
					return err
				}
			}
		}
		frame.Type = "response.completed"
	}
	switch frame.Type {
	case "response.output_text.delta", "response.reasoning_text.delta", "response.reasoning_summary_text.delta":
		event := modelapi.Event{Streaming: s.streaming}
		if frame.Type == "response.output_text.delta" {
			event.Delta = frame.Delta
		} else {
			event.Thinking = frame.Delta
		}
		s.received = true
		s.usage = nil
		return emit(event)
	case "response.output_item.added", "response.output_item.done":
		if frame.Item.Type == "function_call" {
			if frame.OutputIndex == nil {
				return fmt.Errorf("function output has no index")
			}
			return s.lmCall(*frame.OutputIndex, frame.Item, frame.Type == "response.output_item.done")
		}
		if frame.Item.Type != "message" && frame.Item.Type != "reasoning" {
			return fmt.Errorf("unsupported Responses output type")
		}
	case "response.function_call_arguments.delta", "response.function_call_arguments.done":
		index, ok := s.lmItems[frame.ItemID]
		if !ok {
			return fmt.Errorf("arguments refer to an unknown function item")
		}
		draft, err := s.callAt(index)
		if err != nil {
			return err
		}
		if frame.Type == "response.function_call_arguments.delta" {
			return draft.appendArguments(frame.Delta)
		}
		return draft.finalArguments(frame.Arguments)
	case "error":
		return transport.ParseError(0, content, "")
	case "response.completed", "response.failed", "response.incomplete":
		response := frame.Response
		if response.Usage != nil {
			input, output := response.Usage.Input, response.Usage.Output
			if input != nil && (*input < 0 || *input > modelapi.MaxCallTokens) || output != nil && (*output < 0 || *output > modelapi.MaxCallTokens) {
				return fmt.Errorf("invalid provider token usage")
			}
			if input != nil && output != nil {
				s.usage = &modelapi.TokenUsage{Input: *input, Output: *output}
			}
		}
		s.received = true
		if response.Status != "completed" || frame.Type != "response.completed" {
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
				return s.emitFailedUsage(&modelapi.OutputLimitError{Reason: detail}, emit)
			}
			if cause != nil {
				return s.emitFailedUsage(fmt.Errorf("LM Studio: %w", cause), emit)
			}
			return s.emitFailedUsage(fmt.Errorf("LM Studio: %s", detail), emit)
		}
		for index, item := range response.Output {
			if item.Type == "function_call" {
				if err := s.lmCall(index, item, true); err != nil {
					return s.emitFailedUsage(err, emit)
				}
			} else if item.Type != "message" && item.Type != "reasoning" {
				return s.emitFailedUsage(fmt.Errorf("unsupported Responses output type"), emit)
			}
		}
		s.finished = true
	}
	return nil
}
