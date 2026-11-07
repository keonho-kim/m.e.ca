// Purpose: Translate individual provider chunks and their completion and usage state.
// Pattern: Stateful stream decoder.
// Usage: Created per model response by the provider stream reader.
// Related: internal/provider/protocol/stream.go, internal/provider/protocol/native/frame.go, internal/agent/model/model.go, internal/provider/connection/connection.go.

package protocol

import (
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/chat"
	"github.com/keonho-kim/meca/internal/provider/protocol/claude"
	"github.com/keonho-kim/meca/internal/provider/protocol/gemini"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/protocol/ollama"
	"github.com/keonho-kim/meca/internal/provider/protocol/responses"
)

type streamState struct {
	router chat.ReasoningState
	native.State
	responses responses.Decoder
	provider  modelconn.ProviderKind
	claude    claude.Decoder
	gemini    gemini.Decoder
}

func (state *streamState) consume(content []byte, single bool, emit func(modelapi.Event) error) error {
	if state.provider == modelconn.Claude {
		return state.claude.Consume(content, single, &state.State, emit)
	}
	if state.provider == modelconn.Gemini {
		return state.gemini.Consume(content, single, &state.State, emit)
	}
	if modelconn.UsesResponses(state.provider) {
		state.responses.Retain = state.provider == modelconn.OpenAI
		return state.responses.Consume(content, single, &state.State, emit)
	}
	isOllama := state.provider == modelconn.Ollama
	var frame native.Frame
	var err error
	if isOllama {
		frame, err = ollama.Decode(content)
	} else {
		frame, err = chat.Decode(content, single)
	}
	input, output := frame.Input, frame.Output
	if input != nil && (*input < 0 || *input > modelapi.MaxCallTokens) || output != nil && (*output < 0 || *output > modelapi.MaxCallTokens) {
		return fmt.Errorf("invalid provider token usage")
	}
	var observed *modelapi.TokenUsage
	if input != nil && output != nil {
		state.Usage = &modelapi.TokenUsage{Input: *input, Output: *output}
		if state.provider == modelconn.OpenRouter && frame.Cost != nil {
			state.Usage.Cost = frame.Cost
		}
		if err := state.Usage.Check(); err != nil {
			return err
		}
		observed = state.Usage
	}
	if err != nil {
		return state.EmitFailedUsage(err, emit)
	}
	state.Received = true
	message := frame.Message
	if frame.Done != nil {
		state.Finished = *frame.Done
	}
	if frame.Reason != nil {
		if reasonErr := sanitizeReason(*frame.Reason); reasonErr != nil {
			state.Failure = reasonErr
		}
	}

	reasoning := message.Reasoning
	if state.provider == modelconn.LiteLLM || state.provider == modelconn.SGLang {
		reasoning = message.ReasoningContent
	}
	reason := message.Thinking + reasoning
	if state.provider == modelconn.OpenRouter {
		text, err := state.router.Add(message.ReasoningDetails, !single)
		if err != nil {
			return err
		}
		if text != "" || reason == "" {
			reason = text
		}
	}
	if observed == nil && (message.Content != "" || reason != "") {
		state.Usage = nil
	}
	if len(message.Calls) > 0 && string(message.Calls) != "null" && string(message.Calls) != "[]" {
		if err := state.Calls.Add(message.Calls, isOllama, single); err != nil {
			return err
		}
	}
	if message.Content != "" || reason != "" || observed != nil {
		return emit(modelapi.Event{Delta: message.Content, Thinking: reason, Streaming: state.Streaming, Usage: observed})
	}
	return nil
}

func (state *streamState) complete(emit func(modelapi.Event) error) error {
	if state.Failure != nil {
		return state.EmitFailedUsage(state.Failure, emit)
	}
	if !state.Finished || !state.Received {
		return fmt.Errorf("model stream ended before completion")
	}
	calls, err := state.Calls.Complete()
	if err != nil {
		return state.EmitFailedUsage(err, emit)
	}
	continuation := state.Continuation
	if state.router.HasData() {
		continuation, err = state.router.Continuation()
		if err != nil {
			return err
		}
	}
	return emit(modelapi.Event{Done: true, Streaming: state.Streaming, Usage: state.Usage, Calls: calls, ReasoningState: continuation})
}
