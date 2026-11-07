// Purpose: Define Gemini content parts, signed continuation and native function wire values.
// Pattern: Typed GenerateContent boundary.
// Usage: The request adapter and response-owned decoder share these protocol values.
// Related: internal/provider/protocol/gemini_request.go, internal/provider/protocol/gemini/stream.go.
package gemini

import "encoding/json"

type FunctionCall struct {
	ID   string          `json:"id,omitempty"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}
type FunctionResponse struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}
type Blob struct {
	MIME string `json:"mimeType"`
	Data string `json:"data"`
}
type Part struct {
	Text      *string           `json:"text,omitempty"`
	Thought   bool              `json:"thought,omitempty"`
	Signature string            `json:"thoughtSignature,omitempty"`
	Call      *FunctionCall     `json:"functionCall,omitempty"`
	Response  *FunctionResponse `json:"functionResponse,omitempty"`
	Inline    *Blob             `json:"inlineData,omitempty"`
}
type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}
type Declaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"parametersJsonSchema"`
}
type Tool struct {
	Functions []Declaration `json:"functionDeclarations"`
}
type Generation struct {
	MaxOutput   int      `json:"maxOutputTokens"`
	Temperature *float64 `json:"temperature,omitempty"`
	Thinking    any      `json:"thinkingConfig,omitempty"`
}
type Request struct {
	Contents []Content  `json:"contents"`
	System   *Content   `json:"systemInstruction,omitempty"`
	Tools    []Tool     `json:"tools,omitempty"`
	Config   Generation `json:"generationConfig"`
}
type Continuation struct {
	Parts   []Part   `json:"parts"`
	CallIDs []string `json:"call_ids"`
}

// Arguments maps protobuf's omitted or null empty Struct to the engine's object contract.
// The original signed part remains unchanged for replay.
func (c FunctionCall) Arguments() json.RawMessage {
	if len(c.Args) == 0 || string(c.Args) == "null" {
		return json.RawMessage(`{}`)
	}
	return c.Args
}
