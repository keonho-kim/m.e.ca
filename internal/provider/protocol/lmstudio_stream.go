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
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (s *streamState) consumeLM(content []byte, single bool, emit func(modelapi.Event) error) error {
	var frame struct {
		Error    json.RawMessage `json:"error"`
		Type     string          `json:"type"`
		Delta    string          `json:"delta"`
		Response lmResponse      `json:"response"`
		Item     struct {
			Type string `json:"type"`
		} `json:"item"`
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
		for _, item := range frame.Response.Output {
			if item.Type != "message" && item.Type != "reasoning" {
				return fmt.Errorf("native tool execution is not supported")
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
		if frame.Item.Type != "message" && frame.Item.Type != "reasoning" {
			return fmt.Errorf("native tool execution is not supported")
		}
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
		for _, item := range response.Output {
			if item.Type != "message" && item.Type != "reasoning" {
				return s.emitFailedUsage(fmt.Errorf("native tool execution is not supported"), emit)
			}
		}
		if response.Status != "completed" || frame.Type != "response.completed" {
			detail := "response did not complete"
			if response.Incomplete.Reason != "" {
				detail = response.Incomplete.Reason
			}
			if len(response.Error) > 0 && string(response.Error) != "null" {
				detail = transport.ParseError(0, append(append([]byte(`{"error":`), response.Error...), '}'), "").Error()
			}
			if response.Incomplete.Reason == "max_output_tokens" {
				return s.emitFailedUsage(&modelapi.OutputLimitError{Reason: detail}, emit)
			}
			return s.emitFailedUsage(fmt.Errorf("LM Studio: %s", detail), emit)
		}
		s.finished = true
	}
	return nil
}
