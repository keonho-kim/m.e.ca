// Purpose: Keep provider transport DTOs separate from agent domain messages.
// Pattern: Adapter DTOs.
// Usage: Used only within provider request encoding and response parsing.
// Related: internal/provider/httpclient/request.go, internal/provider/httpclient/stream.go, internal/provider/httpclient/chunk.go.

package httpclient

import (
	"encoding/json"
)

type wireMessage struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content"`
	Thinking  string          `json:"thinking,omitempty"`
	Reasoning string          `json:"reasoning_content,omitempty"`
	Calls     json.RawMessage `json:"tool_calls,omitempty"`
}

type ollamaOptions struct {
	NumPredict int `json:"num_predict"`
}

type wireRequest struct {
	Think     *bool          `json:"think,omitempty"`
	Model     string         `json:"model"`
	Messages  []wireMessage  `json:"messages"`
	Stream    bool           `json:"stream"`
	MaxTokens int            `json:"max_tokens,omitempty"`
	Options   *ollamaOptions `json:"options,omitempty"`
}

type wireChunk struct {
	Message     wireMessage `json:"message"`
	Done        *bool       `json:"done"`
	DoneReason  string      `json:"done_reason"`
	PromptCount int         `json:"prompt_eval_count"`
	EvalCount   int         `json:"eval_count"`
	Error       string      `json:"error"`
	Choices     []struct {
		Delta        wireMessage `json:"delta"`
		Message      wireMessage `json:"message"`
		FinishReason *string     `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		Input  int `json:"prompt_tokens"`
		Output int `json:"completion_tokens"`
	} `json:"usage"`
}
