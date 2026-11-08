// Purpose: Mask JSON string tokens without rebuilding containers or changing numeric precision.
// Pattern: Structured secret sanitization boundary.
// Usage: Server detail pages and visible tool arguments sanitize complete JSON before transmission.
// Related: internal/redaction/protected.go, internal/serverapi/detail/document.go.
package redaction

import (
	"encoding/json"

	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func RedactJSON(body []byte, redact func(string) string) ([]byte, error) {
	if !utf8.Valid(body) || !json.Valid(body) {
		return nil, faults.Errorf(diagcatalog.RedactionInvalidCompleteUTFJSON)
	}
	var output []byte
	copied := 0
	err := walkJSONStrings(body, func(start, end int, value string) error {
		masked := redact(value)
		if masked == value {
			return nil
		}
		encoded, err := json.Marshal(masked)
		if err != nil {
			return err
		}
		if output == nil {
			output = make([]byte, 0, len(body))
		}
		output = append(output, body[copied:start]...)
		output = append(output, encoded...)
		copied = end
		return nil
	})
	if err != nil {
		return nil, err
	}
	if output == nil {
		return body, nil
	}
	return append(output, body[copied:]...), nil
}

// Callers validate the complete JSON first. Unescaped strings need no second
// JSON parser; escaped strings retain encoding/json's Unicode semantics.
func walkJSONStrings(body []byte, visit func(start, end int, value string) error) error {
	for i := 0; i < len(body); i++ {
		if body[i] != '"' {
			continue
		}
		start := i
		escaped := false
		for i++; i < len(body); i++ {
			if body[i] == '\\' {
				escaped = true
				i++
			} else if body[i] == '"' {
				break
			}
		}
		var value string
		if escaped {
			if err := json.Unmarshal(body[start:i+1], &value); err != nil {
				return err
			}
		} else {
			value = string(body[start+1 : i])
		}
		if err := visit(start, i+1, value); err != nil {
			return err
		}
	}
	return nil
}
