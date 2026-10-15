// Purpose: Normalize one complete JSON object without extracting it from arbitrary prose.
// Pattern: Pure model-output boundary.
// Usage: Change and completion review parsers share Object.
// Related: internal/agent/conclave/verdict.go, internal/agent/completion/verdict.go.
package reviewjson

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxResponseBytes = 256 * 1024 * 1024

func Object(text string) (string, error) {
	if len(text) > MaxResponseBytes || !utf8.ValidString(text) {
		return "", fmt.Errorf("review response exceeds UTF-8 resource bound")
	}
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		end := strings.IndexByte(text, '\n')
		if end < 0 {
			return "", fmt.Errorf("incomplete JSON fence")
		}
		language := strings.TrimSpace(text[3:end])
		if !strings.EqualFold(language, "json") && language != "" {
			return "", fmt.Errorf("expected JSON fence")
		}
		if !strings.HasSuffix(text, "\n```") {
			return "", fmt.Errorf("JSON fence must enclose the whole response")
		}
		text = strings.TrimSpace(text[end+1 : len(text)-3])
	}
	if !strings.HasPrefix(text, "{") || !json.Valid([]byte(text)) {
		return "", fmt.Errorf("expected one complete JSON object")
	}
	return text, nil
}
