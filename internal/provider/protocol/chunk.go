// Purpose: Translate individual provider chunks and their completion and usage state.
// Pattern: Stateful stream decoder.
// Usage: Created per model response by the provider stream reader.
// Related: internal/provider/protocol/stream.go, internal/provider/protocol/wire.go, internal/agent/model/model.go, internal/provider/connection/connection.go.

package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/transport"
)

type streamState struct {
	calls     map[int]*callDraft
	lmItems   map[string]int
	provider  modelconn.ProviderKind
	usage     *modelapi.TokenUsage
	finished  bool
	streaming bool
	received  bool
	failure   error
}

func (state *streamState) consume(content []byte, single bool, emit func(modelapi.Event) error) error {
	if state.provider == modelconn.LMStudio {
		return state.consumeLM(content, single, emit)
	}
	native := state.provider == modelconn.Ollama
	var chunk wireChunk
	if err := json.Unmarshal(content, &chunk); err != nil {
		return fmt.Errorf("invalid model stream: %w", err)
	}
	input, output := chunk.PromptCount, chunk.EvalCount
	if !native && chunk.Usage != nil {
		input, output = chunk.Usage.Input, chunk.Usage.Output
	}
	if input != nil && (*input < 0 || *input > modelapi.MaxCallTokens) || output != nil && (*output < 0 || *output > modelapi.MaxCallTokens) {
		return fmt.Errorf("invalid provider token usage")
	}
	var observed *modelapi.TokenUsage
	if input != nil && output != nil {
		state.usage = &modelapi.TokenUsage{Input: *input, Output: *output}
		observed = state.usage
	}
	if len(chunk.Error) > 0 && string(chunk.Error) != "null" && string(chunk.Error) != `""` {
		return state.emitFailedUsage(transport.ParseError(0, content, ""), emit)
	}
	state.received = true
	message := chunk.Message
	if native {
		if chunk.Done == nil {
			return fmt.Errorf("missing Ollama done state")
		}
		state.finished = *chunk.Done
		if err := sanitizeReason(chunk.DoneReason); err != nil {
			state.failure = err
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
					state.failure = err
				}
				if single {
					state.finished = true
				}
			}
		} else if chunk.Usage == nil {
			return fmt.Errorf("missing model choices")
		}
	}
	reasoning := message.Reasoning
	if state.provider == modelconn.LiteLLM || state.provider == modelconn.SGLang {
		reasoning = message.ReasoningContent
	}
	reason := message.Thinking + reasoning
	if observed == nil && (message.Content != "" || reason != "") {
		state.usage = nil
	}
	if len(message.Calls) > 0 && string(message.Calls) != "null" && string(message.Calls) != "[]" {
		if err := state.addCalls(message.Calls, native, single); err != nil {
			return err
		}
	}
	if message.Content != "" || reason != "" || observed != nil {
		return emit(modelapi.Event{Delta: message.Content, Thinking: reason, Streaming: state.streaming, Usage: observed})
	}
	return nil
}

func (state *streamState) emitFailedUsage(err error, emit func(modelapi.Event) error) error {
	if state.usage != nil {
		return errors.Join(err, emit(modelapi.Event{Usage: state.usage, Streaming: state.streaming}))
	}
	return err
}

func (state *streamState) complete(emit func(modelapi.Event) error) error {
	if state.failure != nil {
		return state.emitFailedUsage(state.failure, emit)
	}
	if !state.finished || !state.received {
		return fmt.Errorf("Model stream ended before completion")
	}
	calls, err := state.completedCalls()
	if err != nil {
		return state.emitFailedUsage(err, emit)
	}
	return emit(modelapi.Event{Done: true, Streaming: state.streaming, Usage: state.usage, Calls: calls})
}
