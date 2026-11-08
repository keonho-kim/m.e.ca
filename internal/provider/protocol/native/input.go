// Purpose: Carry validated native request values and shared text/image parts across concrete dialect adapters.
// Pattern: Request-only protocol values.
// Usage: Parent conversion borrows engine calls; Claude and Gemini encode these immutable values.
// Related: internal/provider/protocol/native_input.go, internal/provider/protocol/claude/request.go.
package native

import (
	"encoding/json"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/tool"
)

type Input struct {
	Role, Text, ToolCallID, ToolName string
	Calls                            []tool.Call
	State                            json.RawMessage
	Parts                            []ContentPart
}
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}
type ImageURL struct {
	URL string `json:"url"`
}

func DataImage(url string) (string, string, bool) {
	mime, data, ok := strings.Cut(strings.TrimPrefix(url, "data:"), ";base64,")
	return mime, data, ok && strings.HasPrefix(url, "data:") && strings.HasPrefix(mime, "image/")
}
