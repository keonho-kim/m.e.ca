// Purpose: Define native Ollama chat requests and inference options.
// Pattern: Typed request boundary.
// Usage: The protocol encoder supplies its existing messages and observed model context.
// Related: internal/provider/protocol/request.go, internal/provider/protocol/header.go.
package ollama

import "github.com/keonho-kim/meca/internal/provider/protocol/native"

type Options struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumCtx      int      `json:"num_ctx,omitempty"`
	NumPredict  int      `json:"num_predict"`
}
type Request[T any] struct {
	Model           string         `json:"model"`
	Messages        *[]T           `json:"messages,omitempty"`
	Stream          bool           `json:"stream"`
	Think           any            `json:"think,omitempty"`
	Tools           []native.Tool  `json:"tools,omitempty"`
	Options         *Options       `json:"options,omitempty"`
	ChatTemplate    map[string]any `json:"chat_template_kwargs,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	Reasoning       any            `json:"reasoning,omitempty"`
}
