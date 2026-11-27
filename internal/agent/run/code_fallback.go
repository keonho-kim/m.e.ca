// Purpose: Retain complete committed outputs or the longest fitting UTF-8 excerpts for missing returns.
// Pattern: Request-owned projection on the serial execution loop.
// Usage: acceptCodeResult collects successful outputs; finishCode exposes them only for undefined.
// Related: internal/agent/run/code_result.go, internal/tools/codemode/contract.go.
package run

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/tools/codemode"
)

type codeFallback struct {
	Outputs   []json.RawMessage `json:"outputs"`
	Truncated bool              `json:"truncated,omitempty"`
	bytes     int
}

func (f *codeFallback) retain(output codeObservation) error {
	remaining := codemode.MaxFallbackBytes - 2 - f.bytes // Array brackets.
	if len(f.Outputs) > 0 {
		remaining-- // Separator before the next output.
	}
	if remaining <= 0 {
		f.Truncated = true
		return nil
	}
	original := output.Value
	output.Value = nil
	metadata, err := json.Marshal(output)
	if err != nil {
		return err
	}
	valueBudget := remaining - len(metadata) + len("null")
	if compactJSONFits(original, valueBudget) {
		output.Value = original
	} else {
		f.Truncated = true
		const emptyExcerpt = `{"excerpt":"","truncated":true}`
		end := jsonStringPrefix(original, valueBudget-len(emptyExcerpt))
		if end <= 0 {
			return nil
		}
		output.Value, err = json.Marshal(struct {
			Excerpt   string `json:"excerpt"`
			Truncated bool   `json:"truncated"`
		}{string(original[:end]), true})
		if err != nil {
			return err
		}
	}
	body, err := json.Marshal(output)
	if err != nil {
		return err
	}
	if len(f.Outputs) > 0 {
		f.bytes++
	}
	f.bytes += len(body)
	f.Outputs = append(f.Outputs, body)
	return nil
}

// Inputs are validated and redacted before admission. Count encoding/json's
// compact, HTML-escaped form before allocating a potentially oversized result.
func compactJSONFits(body []byte, budget int) bool {
	inString, escaped := false, false
	for i := 0; i < len(body); i++ {
		c := body[i]
		if !inString && (c == ' ' || c == '\n' || c == '\r' || c == '\t') {
			continue
		}
		size := 1
		if c == '<' || c == '>' || c == '&' {
			size = 6
		} else if c == 0xe2 && i+2 < len(body) && body[i+1] == 0x80 && (body[i+2] == 0xa8 || body[i+2] == 0xa9) {
			size = 6 // U+2028 and U+2029 become Unicode escapes.
			i += 2
		}
		budget -= size
		if budget < 0 {
			return false
		}
		if escaped {
			escaped = false
		} else if c == '\\' && inString {
			escaped = true
		} else if c == '"' {
			inString = !inString
		}
	}
	return budget >= 0
}

// Count actual JSON string escapes and stop only at a complete UTF-8 character.
// Quotes around the excerpt are already included in its empty JSON envelope.
func jsonStringPrefix(body []byte, budget int) int {
	end := 0
	for end < len(body) {
		r, width := utf8.DecodeRune(body[end:])
		encoded := width
		switch r {
		case '"', '\\', '\b', '\f', '\n', '\r', '\t':
			encoded = 2
		case '<', '>', '&', '\u2028', '\u2029':
			encoded = 6
		default:
			if r < 0x20 {
				encoded = 6
			}
		}
		if encoded > budget {
			break
		}
		budget -= encoded
		end += width
	}
	return end
}
