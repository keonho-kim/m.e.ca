// Purpose: Reassemble bounded text and Tool Calls from parsed provider chunks.
// Pattern: Stateful stream decoder.
// Usage: Created per model response by the provider stream reader.
// Related: internal/provider/httpclient/stream.go, internal/provider/httpclient/wire.go, internal/agent/model.go.

package httpclient

import (
	"encoding/json"
	"fmt"

	"github.com/keonho-kim/v-code/internal/agent"
)

type streamState struct {
	bytes        int
	inputTokens  int
	outputTokens int
	finished     bool
	streaming    bool
	received     bool
}

func (state *streamState) consume(content []byte, native, single bool, emit func(agent.ModelEvent) error) error {
	var chunk wireChunk
	if err := json.Unmarshal(content, &chunk); err != nil {
		return fmt.Errorf("invalid model stream: %w", err)
	}
	if chunk.Error != "" {
		return fmt.Errorf("Model server reported a generation error")
	}
	if chunk.PromptCount < 0 || chunk.EvalCount < 0 || chunk.Usage.Input < 0 || chunk.Usage.Output < 0 {
		return fmt.Errorf("invalid negative provider token usage")
	}
	state.received = true
	message := chunk.Message
	if native {
		if chunk.Done == nil {
			return fmt.Errorf("missing Ollama done state")
		}
		state.inputTokens, state.outputTokens = chunk.PromptCount, chunk.EvalCount
		state.finished = *chunk.Done
		if err := sanitizeReason(chunk.DoneReason); err != nil {
			return err
		}
	} else {
		if len(chunk.Choices) > 1 {
			return fmt.Errorf("multiple model choices are not supported")
		}
		if len(chunk.Choices) == 1 {
			choice := chunk.Choices[0]
			message = choice.Delta
			if single {
				message = choice.Message
			}
			if choice.FinishReason != nil {
				if err := sanitizeReason(*choice.FinishReason); err != nil {
					return err
				}
				if single {
					state.finished = true
				}
			}
		} else if chunk.Usage.Input == 0 && chunk.Usage.Output == 0 {
			return fmt.Errorf("missing model choices")
		}
		if chunk.Usage.Input > 0 {
			state.inputTokens = chunk.Usage.Input
		}
		if chunk.Usage.Output > 0 {
			state.outputTokens = chunk.Usage.Output
		}
	}
	reason := message.Thinking + message.Reasoning
	state.bytes += len(message.Content) + len(reason)
	if state.bytes > agent.MaxResponseBytes {
		return fmt.Errorf("model output exceeds size limit")
	}
	if len(message.Calls) > 0 && string(message.Calls) != "null" && string(message.Calls) != "[]" {
		return fmt.Errorf("native JSON tool calls are not supported; reply using the staged text dialogue")
	}
	if !single && (message.Content != "" || reason != "" || len(message.Calls) > 0) {
		return emit(agent.ModelEvent{Delta: message.Content, Thinking: reason, Streaming: state.streaming})
	}
	if single && (message.Content != "" || reason != "") {
		return emit(agent.ModelEvent{Delta: message.Content, Thinking: reason})
	}
	return nil
}

func (state *streamState) complete(emit func(agent.ModelEvent) error) error {
	if !state.finished || !state.received {
		return fmt.Errorf("Model stream ended before completion")
	}
	return emit(agent.ModelEvent{Done: true, Streaming: state.streaming, InputTokens: state.inputTokens, OutputTokens: state.outputTokens})
}
