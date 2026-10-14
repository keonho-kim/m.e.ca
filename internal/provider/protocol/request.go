// Purpose: Encode the selected provider request dialect without performing HTTP I/O.
// Pattern: Simple Module.
// Usage: The generation adapter calls Encode after validating model controls and context.
// Related: internal/provider/protocol/wire.go, internal/provider/protocol/lmstudio.go.
package protocol

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

func Encode(kind modelconn.ProviderKind, baseURL string, info *modelconn.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, []byte, error) {
	payload := wireRequest{Model: request.ModelID, Messages: messages, Stream: true, Think: controls.Think, ChatTemplate: controls.Template, ReasoningEffort: controls.ReasoningEffort}
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
	content, err := json.Marshal(payload)
	if kind == modelconn.LMStudio {
		endpoint = baseURL + "/responses"
		content, err = lmPayload(request.ModelID, messages, request.MaxOutput, controls.ReasoningEffort, request.Temperature)
	}
	if err != nil {
		return "", nil, err
	}
	return endpoint, content, nil
}
