// Purpose: Encode typed text Chat Completions and measured task usage in OpenAI wire format.
// Pattern: Pure transport projection.
// Usage: Nonstreaming replies and SSE share identities and finish reasons.
// Related: internal/serverapi/openai/stream.go, internal/commands/server/response.go.
package openai

import "github.com/keonho-kim/meca/internal/commands/server"

type chatTokens struct {
	Input  int64 `json:"prompt_tokens"`
	Output int64 `json:"completion_tokens"`
	Total  int64 `json:"total_tokens"`
}
type chatDelta struct {
	Role    string  `json:"role,omitempty"`
	Content *string `json:"content,omitempty"`
}
type chatStreamChoice struct {
	Index    int       `json:"index"`
	Delta    chatDelta `json:"delta"`
	Finish   *string   `json:"finish_reason"`
	Logprobs any       `json:"logprobs"`
}
type chatStream struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []chatStreamChoice `json:"choices"`
	Usage   *chatTokens        `json:"usage"`
}

func chatUsage(value server.Response) *chatTokens {
	if value.Usage == nil {
		return nil
	}
	u := value.Usage
	return &chatTokens{u.Input, u.Output, u.Input + u.Output}
}
func finishReason(value server.Response) string {
	if value.Status == "incomplete" {
		return "length"
	}
	return "stop"
}
func chatID(value server.Response) string { return "chatcmpl_" + value.ID[5:] }
func chatValue(value server.Response, text string) any {
	return struct {
		ID      string        `json:"id"`
		Object  string        `json:"object"`
		Created int64         `json:"created"`
		Model   string        `json:"model"`
		Choices [1]chatChoice `json:"choices"`
		Usage   *chatTokens   `json:"usage"`
	}{chatID(value), "chat.completion", value.Created, "meca", [1]chatChoice{{Message: chatMessage{Role: "assistant", Content: text}, Finish: finishReason(value)}}, chatUsage(value)}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Refusal any    `json:"refusal"`
}
type chatChoice struct {
	Index    int         `json:"index"`
	Message  chatMessage `json:"message"`
	Finish   string      `json:"finish_reason"`
	Logprobs any         `json:"logprobs"`
}

func chatChunk(id string, created int64, delta chatDelta, finish *string) chatStream {
	return chatStream{ID: id, Object: "chat.completion.chunk", Created: created, Model: "meca", Choices: []chatStreamChoice{{Delta: delta, Finish: finish}}}
}
