// Purpose: Supply native complete responses for cross-provider configuration contract tests.
// Pattern: Local HTTP fixture.
// Usage: Default-control and sampling tests keep their assertions independent of response dialect.
// Related: internal/provider/httpclient/temperature_test.go, internal/provider/httpclient/alias_controls_test.go.
package httpclient

import (
	"encoding/json"
	"net/http"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

func writeProviderSuccess(w http.ResponseWriter, kind connection.ProviderKind, text string) {
	var value any
	switch kind {
	case connection.Ollama:
		value = map[string]any{"message": map[string]string{"content": text}, "done": true}
	case connection.LMStudio, connection.OpenAI:
		value = map[string]any{"status": "completed", "output": []any{map[string]any{"type": "message", "content": []any{map[string]string{"type": "output_text", "text": text}}}}}
	case connection.Claude:
		value = map[string]any{"type": "message", "stop_reason": "end_turn", "content": []any{map[string]string{"type": "text", "text": text}}}
	case connection.Gemini:
		value = map[string]any{"candidates": []any{map[string]any{"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]string{"text": text}}}}}}
	default:
		value = map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": text}, "finish_reason": "stop"}}}
	}
	_ = json.NewEncoder(w).Encode(value)
}
