// Purpose: Translate individual provider chunks and their completion and usage state.
// Pattern: Stateful stream decoder.
// Usage: Created per model response by the provider stream reader.
// Related: internal/provider/protocol/stream.go, internal/provider/protocol/native/frame.go, internal/agent/model/model.go, internal/provider/connection/connection.go.

package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
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
		return faults.Errorf(diagcatalog.ProtocolInvalidProviderTokenUsage)
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
	opaqueProgress := false
	if state.provider == modelconn.OpenRouter {
		previous := state.router.Progress()
		text, err := state.router.Add(message.ReasoningDetails, !single)
		if err != nil {
			return err
		}
		if text != "" || reason == "" {
			reason = text
		}
		opaqueProgress = state.router.Progress() != previous
	}
	if observed == nil && (message.Content != "" || reason != "") {
		state.Usage = nil
	}
	beforeCalls := state.Calls.Bytes()
	if len(message.Calls) > 0 && string(message.Calls) != "null" && string(message.Calls) != "[]" {
		if err := state.Calls.Add(message.Calls, isOllama, single); err != nil {
			return err
		}
	}
	progress := state.Calls.Bytes() > beforeCalls || opaqueProgress
	if message.Content != "" || reason != "" || observed != nil || progress {
		return emit(modelapi.Event{Progress: progress, Delta: message.Content, Thinking: reason, Streaming: state.Streaming, Usage: observed})
	}
	return nil
}

func (state *streamState) complete(emit func(modelapi.Event) error) error {
	if state.Failure != nil {
		return state.EmitFailedUsage(state.Failure, emit)
	}
	if !state.Finished || !state.Received {
		return faults.Transient(faults.Errorf(diagcatalog.ProtocolModelStreamEndedBeforeCompletion))
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
