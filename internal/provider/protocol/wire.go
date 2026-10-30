// Purpose: Keep provider transport DTOs separate from agent domain messages.
// Pattern: Adapter DTOs.
// Usage: Used only within provider request encoding and response parsing.
// Related: internal/provider/protocol/request.go, internal/provider/protocol/stream.go, internal/provider/protocol/chunk.go.

package protocol

import (
	"encoding/json"
)

type responseMessage struct {
	ReasoningDetails []json.RawMessage `json:"reasoning_details,omitempty"`
	ToolCallID       string            `json:"tool_call_id,omitempty"`
	ToolName         string            `json:"tool_name,omitempty"`
	Role             string            `json:"role,omitempty"`
	Content          string            `json:"content"`
	Thinking         string            `json:"thinking,omitempty"`
	Reasoning        string            `json:"reasoning,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	Calls            json.RawMessage   `json:"tool_calls,omitempty"`
}

type ollamaOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumCtx      int      `json:"num_ctx,omitempty"`
	NumPredict  int      `json:"num_predict"`
}

type wireRequest struct {
	Reasoning       *routerReasoning `json:"reasoning,omitempty"`
	Tools           []Tool           `json:"tools,omitempty"`
	ToolChoice      string           `json:"tool_choice,omitempty"`
	ParallelCalls   *bool            `json:"parallel_tool_calls,omitempty"`
	Temperature     *float64         `json:"temperature,omitempty"`
	StreamOptions   *streamOptions   `json:"stream_options,omitempty"`
	ReasoningEffort string           `json:"reasoning_effort,omitempty"`
	Think           any              `json:"think,omitempty"`
	Model           string           `json:"model"`
	Messages        []Message        `json:"messages"`
	Stream          bool             `json:"stream"`
	MaxTokens       int              `json:"max_tokens,omitempty"`
	Options         *ollamaOptions   `json:"options,omitempty"`
	ChatTemplate    map[string]any   `json:"chat_template_kwargs,omitempty"`
}

type routerReasoning struct {
	Enabled bool   `json:"enabled"`
	Effort  string `json:"effort,omitempty"`
}

type streamOptions struct {
	IncludeUsage    bool `json:"include_usage"`
	ContinuousUsage bool `json:"continuous_usage_stats,omitempty"`
}

type wireUsage struct {
	Input  *int64   `json:"prompt_tokens"`
	Output *int64   `json:"completion_tokens"`
	Cost   *float64 `json:"cost"`
}

type wireChunk struct {
	Message     responseMessage `json:"message"`
	Done        *bool           `json:"done"`
	DoneReason  string          `json:"done_reason"`
	PromptCount *int64          `json:"prompt_eval_count"`
	EvalCount   *int64          `json:"eval_count"`
	Error       json.RawMessage `json:"error"`
	Choices     []struct {
		Delta        responseMessage `json:"delta"`
		Message      responseMessage `json:"message"`
		FinishReason *string         `json:"finish_reason"`
	} `json:"choices"`
	Usage *wireUsage `json:"usage"`
}
