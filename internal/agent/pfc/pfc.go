// Purpose: Parse and validate individual Progressive Field Completion literal fields.
// Pattern: Pure Function (model trust boundary).
// Usage: Used by field collection, provider diagnostics and the local tool argument boundary.
// Related: internal/agent/tool/execution.go, internal/agent/pfc/pfc_test.go.
package pfc

import (
	"bytes"
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ParseArgument(p toolapi.Parameter, text string) (json.RawMessage, error) {
	if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return nil, fmt.Errorf("%s has invalid encoding", p.Name)
	}
	if p.Kind == toolapi.Integer {
		text = strings.TrimSpace(text)
		if text == "" && !p.Required {
			text = "0"
		}
		value, err := strconv.Atoi(text)
		if err != nil || value < p.Min || value > p.Max {
			return nil, fmt.Errorf("%s requires one integer from %d to %d", p.Name, p.Min, p.Max)
		}
		return json.Marshal(value)
	}
	if p.Kind == toolapi.LiteralPath {
		// A text response may end its single path line with LF or CRLF.
		text = strings.TrimRight(text, "\r\n")
		trimmed := strings.TrimSpace(text)
		if strings.ContainsAny(text, "\r\n") {
			return nil, fmt.Errorf("%s requires one plain path line", p.Name)
		}
		if json.Valid([]byte(trimmed)) && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "\"")) {
			return nil, fmt.Errorf("%s requires one literal path, not a JSON wrapper", p.Name)
		}
	}
	if p.Kind == toolapi.LineAnchor {
		text = strings.TrimSpace(text)
		if _, err := toolapi.ParseAnchor(text); err != nil {
			return nil, err
		}
	}
	if p.Kind != toolapi.Text && p.Kind != toolapi.LiteralPath && p.Kind != toolapi.LineAnchor {
		return nil, fmt.Errorf("unsupported parameter kind")
	}
	if len(text) > p.MaxBytes || !p.AllowEmpty && strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("%s must be valid literal text within %d bytes", p.Name, p.MaxBytes)
	}
	return json.Marshal(text)
}
func ValidateArgument(p toolapi.Parameter, value json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fmt.Errorf("%s cannot be null", p.Name)
	}
	var text string
	if p.Kind == toolapi.Integer {
		var number int
		if err := json.Unmarshal(value, &number); err != nil {
			return err
		}
		text = strconv.Itoa(number)
	} else {
		if err := json.Unmarshal(value, &text); err != nil {
			return err
		}
	}
	_, err := ParseArgument(p, text)
	return err
}
