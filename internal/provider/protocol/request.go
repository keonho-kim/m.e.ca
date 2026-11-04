// Purpose: Compose the selected typed API request without performing HTTP I/O.
// Pattern: Provider request adapter.
// Usage: Full encoding and replayable headers share the same dialect constructors.
// Related: internal/provider/protocol/chat/request.go, internal/provider/protocol/ollama/request.go, internal/provider/protocol/responses/request.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/chat"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/protocol/ollama"
	"github.com/keonho-kim/meca/internal/provider/protocol/responses"
)

func Encode(kind modelconn.ProviderKind, baseURL string, info *modelconn.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, []byte, error) {
	if kind == modelconn.LMStudio {
		input, err := responseInput(messages)
		if err != nil {
			return "", nil, err
		}
		body := responses.Header(request.ModelID, request.MaxOutput, controls.ReasoningEffort, request.Temperature, request.Tools)
		body.Input = &input
		content, err := wireJSON.Marshal(body)
		return baseURL + "/responses", content, err
	}
	if kind == modelconn.Ollama {
		body := ollamaPayload(info, request, controls)
		body.Messages = &messages
		content, err := wireJSON.Marshal(body)
		return baseURL + "/chat", content, err
	}
	endpoint, body := chatPayload(kind, baseURL, request, controls)
	body.Messages = &messages
	content, err := wireJSON.Marshal(body)
	return endpoint, content, err
}
func chatPayload(kind modelconn.ProviderKind, baseURL string, request modelapi.Request, controls Controls) (string, chat.Request[Message]) {
	body := chat.Request[Message]{Model: request.ModelID, Stream: true, Think: controls.Think, ChatTemplate: controls.Template, ReasoningEffort: controls.ReasoningEffort, Reasoning: controls.Reasoning, MaxTokens: request.MaxOutput, Temperature: request.Temperature, Tools: native.Tools(request.Tools)}
	if len(body.Tools) > 0 {
		parallel := true
		body.ToolChoice, body.ParallelCalls = "auto", &parallel
	}
	body.StreamOptions = &chat.StreamOptions{IncludeUsage: true, ContinuousUsage: kind == modelconn.VLLM || kind == modelconn.SGLang}
	return baseURL + "/chat/completions", body
}
func ollamaPayload(info *modelconn.ModelInfo, request modelapi.Request, controls Controls) ollama.Request[Message] {
	body := ollama.Request[Message]{Model: request.ModelID, Stream: true, Think: controls.Think, ChatTemplate: controls.Template, ReasoningEffort: controls.ReasoningEffort, Tools: native.Tools(request.Tools), Options: &ollama.Options{NumPredict: request.MaxOutput, Temperature: request.Temperature}}
	if info != nil {
		body.Options.NumCtx = info.ContextWindow
	}
	if controls.Reasoning != nil {
		body.Reasoning = controls.Reasoning
	}
	return body
}
