// Purpose: Encode stateless LM Studio Responses requests with full ordered role history.
// Pattern: Provider wire adapter.
// Usage: Generate sends this payload to /v1/responses and uses the shared retry lifecycle.
// Related: internal/provider/protocol/lmstudio_stream.go, internal/provider/protocol/request.go.
package protocol

import "encoding/json"

func lmPayload(model string, messages []Message, maxOutput int, effort string, temperature *float64) ([]byte, error) {
	body := struct {
		Temperature *float64          `json:"temperature,omitempty"`
		Model       string            `json:"model"`
		Input       []Message         `json:"input"`
		Stream      bool              `json:"stream"`
		Store       bool              `json:"store"`
		Truncation  string            `json:"truncation"`
		MaxOutput   int               `json:"max_output_tokens"`
		Reasoning   map[string]string `json:"reasoning,omitempty"`
	}{Temperature: temperature, Model: model, Input: messages, Stream: true, Store: false, Truncation: "disabled", MaxOutput: maxOutput}
	if effort != "" {
		body.Reasoning = map[string]string{"effort": effort}
	}
	return json.Marshal(body)
}
