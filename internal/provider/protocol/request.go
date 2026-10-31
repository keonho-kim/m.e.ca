// Purpose: Encode the selected provider request dialect without performing HTTP I/O.
// Pattern: Simple Module.
// Usage: The generation adapter calls Encode after validating model controls and context.
// Related: internal/provider/protocol/wire.go, internal/provider/protocol/lmstudio.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
)

func Encode(kind modelconn.ProviderKind, baseURL string, info *modelconn.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, []byte, error) {
	if kind == modelconn.LMStudio {
		content, err := lmPayload(request.ModelID, messages, request.MaxOutput, controls.ReasoningEffort, request.Temperature, request.Tools)
		if err != nil {
			return "", nil, err
		}
		return baseURL + "/responses", content, nil
	}
	endpoint, payload := chatPayload(kind, baseURL, info, request, controls)
	payload.Messages = &messages
	content, err := wireJSON.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	return endpoint, content, nil
}

func chatPayload(kind modelconn.ProviderKind, baseURL string, info *modelconn.ModelInfo, request modelapi.Request, controls Controls) (string, wireRequest) {
	payload := wireRequest{Model: request.ModelID, Stream: true, Think: controls.Think, ChatTemplate: controls.Template, ReasoningEffort: controls.ReasoningEffort}
	payload.Reasoning = controls.Reasoning
	payload.Tools = Tools(request.Tools)
	if len(payload.Tools) > 0 && kind != modelconn.Ollama {
		parallel := true
		payload.ToolChoice, payload.ParallelCalls = "auto", &parallel
	}
	endpoint := baseURL + "/chat/completions"
	if kind == modelconn.Ollama {
		endpoint = baseURL + "/chat"
		payload.Options = &ollamaOptions{NumPredict: request.MaxOutput, Temperature: request.Temperature}
		if info != nil {
			payload.Options.NumCtx = info.ContextWindow
		}
	} else {
		payload.MaxTokens = request.MaxOutput
		payload.Temperature = request.Temperature
		payload.StreamOptions = &streamOptions{IncludeUsage: true, ContinuousUsage: kind == modelconn.VLLM || kind == modelconn.SGLang}
	}
	return endpoint, payload
}
