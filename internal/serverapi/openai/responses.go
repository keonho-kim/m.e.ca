// Purpose: Project durable task state and public assistant text into typed Responses objects.
// Pattern: Pure transport projection.
// Usage: Creation, lookup, cancellation and terminal SSE events share responseValue.
// Related: internal/serverapi/openai/handler.go, internal/serverapi/openai/stream.go.
package openai

import "github.com/keonho-kim/meca/internal/commands/server"

type outputText struct {
	Type        string `json:"type"`
	Text        any    `json:"text"`
	Annotations []any  `json:"annotations"`
	Logprobs    []any  `json:"logprobs"`
}
type messageValue struct {
	ID      string       `json:"id"`
	Type    string       `json:"type"`
	Status  string       `json:"status"`
	Role    string       `json:"role"`
	Content []outputText `json:"content"`
}
type responseTokens struct {
	Input  int64 `json:"input_tokens"`
	Output int64 `json:"output_tokens"`
	Total  int64 `json:"total_tokens"`
}
type responseFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type incompleteDetails struct {
	Reason string `json:"reason"`
}
type textFormat struct {
	Type string `json:"type"`
}
type textOptions struct {
	Format textFormat `json:"format"`
}
type responseObject struct {
	ID           string             `json:"id"`
	Object       string             `json:"object"`
	Created      int64              `json:"created_at"`
	Completed    *int64             `json:"completed_at"`
	Status       string             `json:"status"`
	Error        *responseFailure   `json:"error"`
	Incomplete   *incompleteDetails `json:"incomplete_details"`
	Model        string             `json:"model"`
	Output       []messageValue     `json:"output"`
	Usage        *responseTokens    `json:"usage"`
	Previous     *string            `json:"previous_response_id"`
	Store        bool               `json:"store"`
	Background   bool               `json:"background"`
	Tools        []any              `json:"tools"`
	ToolChoice   string             `json:"tool_choice"`
	Parallel     bool               `json:"parallel_tool_calls"`
	Instructions any                `json:"instructions"`
	MaxOutput    any                `json:"max_output_tokens"`
	Temperature  any                `json:"temperature"`
	TopP         any                `json:"top_p"`
	Metadata     struct{}           `json:"metadata"`
	Text         textOptions        `json:"text"`
	Truncation   string             `json:"truncation"`
}

func messageID(value server.Response) string { return "msg_" + value.ID[5:] }
func textPart(text any) outputText {
	return outputText{Type: "output_text", Text: text, Annotations: []any{}, Logprobs: []any{}}
}
func outputMessage(value server.Response, text any, present bool, status string) messageValue {
	content := []outputText{}
	if present {
		content = append(content, textPart(text))
	}
	return messageValue{messageID(value), "message", status, "assistant", content}
}
func responseValue(value server.Response, text string) responseObject {
	return responseWithText(value, text, text != "")
}
func responseWithText(value server.Response, text any, present bool) responseObject {
	r := responseObject{ID: value.ID, Object: "response", Created: value.Created, Status: value.Status, Model: "meca", Output: []messageValue{}, Store: value.Stored, Background: value.Background, Tools: []any{}, ToolChoice: "none", Text: textOptions{textFormat{"text"}}, Truncation: "disabled"}
	if present {
		status := "incomplete"
		if value.Status == "completed" {
			status = "completed"
		}
		r.Output = append(r.Output, outputMessage(value, text, true, status))
	}
	if u := value.Usage; u != nil {
		r.Usage = &responseTokens{u.Input, u.Output, u.Input + u.Output}
	}
	if value.Previous != "" {
		r.Previous = &value.Previous
	}
	if value.Completed > 0 {
		r.Completed = &value.Completed
	}
	if value.Status == "failed" {
		r.Error = &responseFailure{value.Code, value.Error}
	}
	if value.Status == "incomplete" {
		r.Incomplete = &incompleteDetails{"meca_budget_exhausted"}
	}
	return r
}
