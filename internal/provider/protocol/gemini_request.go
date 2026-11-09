// Purpose: Compose Gemini whole-request bodies through its concrete dialect encoder.
// Pattern: Request composition adapter.
// Usage: Full and cached generation preserve the same Gemini transformations and endpoint.
// Related: internal/provider/protocol/gemini/request.go, internal/provider/protocol/gemini_cloud.go.
package protocol

import (
	"net/url"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/gemini"
)

func geminiRequest(base string, request modelapi.Request, messages []Message, controls Controls) (string, []byte, error) {
	body := geminiBase(request, messages, controls)
	wireIDs := map[string]string{}
	for _, message := range messages {
		role, parts, err := geminiMessage(message, wireIDs)
		if err != nil {
			return "", nil, err
		}
		if len(parts) == 0 {
			continue
		}
		last := len(body.Contents) - 1
		if last >= 0 && body.Contents[last].Role == role {
			body.Contents[last].Parts = append(body.Contents[last].Parts, parts...)
		} else {
			body.Contents = append(body.Contents, gemini.Content{Role: role, Parts: parts})
		}
	}
	encoded, err := wireJSON.Marshal(body)
	return geminiEndpoint(base, request.ModelID), encoded, err
}

func geminiEndpoint(base, model string) string {
	return base + "/models/" + url.PathEscape(strings.TrimPrefix(model, "models/")) + ":streamGenerateContent?alt=sse"
}

func geminiBase(request modelapi.Request, messages []Message, controls Controls) gemini.Request {
	var system []string
	for _, message := range messages {
		if message.Role == "system" {
			text, _ := message.Content.(string)
			system = append(system, text)
		}
	}
	return gemini.Base(request, system, controls.Think, controls.OmitThinkingSummary)
}
func geminiMessage(message Message, wireIDs map[string]string) (string, []gemini.Part, error) {
	input, err := nativeMessage(message)
	if err != nil {
		return "", nil, err
	}
	return gemini.EncodeMessage(input, wireIDs)
}
