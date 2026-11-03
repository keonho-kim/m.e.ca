// Purpose: Define Chat Completions requests and explicit reasoning controls.
// Pattern: Typed request boundary.
// Usage: The protocol encoder supplies its existing cached message DTOs without copying them.
// Related: internal/provider/protocol/request.go, internal/provider/protocol/header.go.
package chat

import "github.com/keonho-kim/meca/internal/provider/protocol/native"

type Request[T any] struct {
	Reasoning       *Reasoning     `json:"reasoning,omitempty"`
	Tools           []native.Tool  `json:"tools,omitempty"`
	ToolChoice      string         `json:"tool_choice,omitempty"`
	ParallelCalls   *bool          `json:"parallel_tool_calls,omitempty"`
	Temperature     *float64       `json:"temperature,omitempty"`
	StreamOptions   *StreamOptions `json:"stream_options,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	Think           any            `json:"think,omitempty"`
	Model           string         `json:"model"`
	Messages        *[]T           `json:"messages,omitempty"`
	Stream          bool           `json:"stream"`
	MaxTokens       int            `json:"max_tokens,omitempty"`
	ChatTemplate    map[string]any `json:"chat_template_kwargs,omitempty"`
}

type Reasoning struct {
	Enabled bool   `json:"enabled"`
	Effort  string `json:"effort,omitempty"`
}

type StreamOptions struct {
	IncludeUsage    bool `json:"include_usage"`
	ContinuousUsage bool `json:"continuous_usage_stats,omitempty"`
}
