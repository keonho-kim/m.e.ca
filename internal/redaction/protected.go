// Purpose: Reject protected content before opaque or JSON-encoded native state reaches storage or models.
// Pattern: Structured secret validation boundary.
// Usage: Agent generation and context projection validate state and nested JSON argument strings.
// Related: internal/redaction/json.go, internal/agent/run/generate.go, internal/agent/context/projection.go.
package redaction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// CheckProtected inspects JSON string values, including JSON serialized inside a
// string, without changing signed provider bytes. Non-JSON opaque state still
// receives the normal text check. Encrypted bytes remain provider-owned.
func CheckProtected(body []byte, redact func(string) string) error {
	if text := string(body); redact(text) != text {
		return fmt.Errorf("native state contains protected content")
	}
	return checkJSONProtected(body, redact, 0)
}

func checkJSONProtected(body []byte, redact func(string) string, depth int) error {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' && body[0] != '[' && body[0] != '"' {
		return nil
	}
	if depth > 64 {
		return fmt.Errorf("native state JSON nesting exceeds limit")
	}
	if !json.Valid(body) {
		return nil
	}
	if !utf8.Valid(body) {
		return fmt.Errorf("invalid complete UTF-8 JSON")
	}
	return walkJSONStrings(body, func(_, _ int, value string) error {
		if redact(value) != value {
			return fmt.Errorf("native state contains protected content")
		}
		nested := strings.TrimSpace(value)
		if len(nested) == 0 || nested[0] != '{' && nested[0] != '[' && nested[0] != '"' {
			return nil
		}
		return checkJSONProtected([]byte(nested), redact, depth+1)
	})
}
