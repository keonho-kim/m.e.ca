// Purpose: Define stateless Responses headers and native function declarations.
// Pattern: Typed provider request boundary.
// Usage: The protocol encoder attaches its ordered input without retaining another conversation.
// Related: internal/provider/protocol/response_input.go, internal/provider/protocol/responses/stream.go.
package responses

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

type Request struct {
	Tools       []map[string]any  `json:"tools,omitempty"`
	ToolChoice  string            `json:"tool_choice,omitempty"`
	Parallel    *bool             `json:"parallel_tool_calls,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	Model       string            `json:"model"`
	Input       *[]map[string]any `json:"input,omitempty"`
	Stream      bool              `json:"stream"`
	Store       bool              `json:"store"`
	Truncation  string            `json:"truncation"`
	MaxOutput   int               `json:"max_output_tokens"`
	Reasoning   map[string]string `json:"reasoning,omitempty"`
}

func Header(model string, maxOutput int, effort string, temperature *float64, definitions []toolapi.Definition) Request {
	body := Request{Temperature: temperature, Model: model, Stream: true, Store: false, Truncation: "disabled", MaxOutput: maxOutput}
	for _, tool := range native.Tools(definitions) {
		body.Tools = append(body.Tools, map[string]any{"type": "function", "name": tool.Function.Name, "description": tool.Function.Description, "parameters": tool.Function.Parameters})
	}
	if len(body.Tools) > 0 {
		parallel := true
		body.Parallel, body.ToolChoice = &parallel, "auto"
	}
	if effort != "" {
		body.Reasoning = map[string]string{"effort": effort}
	}
	return body
}
