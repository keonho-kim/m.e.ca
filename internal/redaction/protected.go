// Purpose: Reject protected content before opaque or JSON-encoded native state reaches storage or models.
// Pattern: Structured secret validation boundary.
// Usage: Agent generation and context projection validate state and nested JSON argument strings.
// Related: internal/redaction/json.go, internal/agent/run/generate.go, internal/agent/context/projection.go.
package redaction

import (
	"bytes"
	"encoding/json"

	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// CheckProtected inspects JSON string values, including JSON serialized inside a
	// string, without changing signed provider bytes. Non-JSON opaque state still
	// receives the normal text check. Encrypted bytes remain provider-owned.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func CheckProtected(body []byte, redact func(string) string) error {
	if text := string(body); redact(text) != text {
		return faults.Errorf(diagcatalog.RedactionNativeStateContainsProtectedContent)
	}
	return checkJSONProtected(body, redact, 0)
}

func checkJSONProtected(body []byte, redact func(string) string, depth int) error {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' && body[0] != '[' && body[0] != '"' {
		return nil
	}
	if depth > 64 {
		return faults.Errorf(diagcatalog.RedactionNativeStateJSONNestingExceedsLimit)
	}
	if !json.Valid(body) {
		return nil
	}
	if !utf8.Valid(body) {
		return faults.Errorf(diagcatalog.RedactionInvalidCompleteUTFJSON)
	}
	return walkJSONStrings(body, func(_, _ int, value string) error {
		if redact(value) != value {
			return faults.Errorf(diagcatalog.RedactionNativeStateContainsProtectedContent)
		}
		nested := strings.TrimSpace(value)
		if len(nested) == 0 || nested[0] != '{' && nested[0] != '[' && nested[0] != '"' {
			return nil
		}
		return checkJSONProtected([]byte(nested), redact, depth+1)
	})
}
