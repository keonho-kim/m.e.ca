// Purpose: Validate helper envelopes while borrowing the data range from an immutable capture buffer.
// Pattern: Standard JSON token parser with offsets into caller-owned bytes.
// Usage: invoke retains capture storage until the returned captured value is closed.
// Related: internal/tools/browser/capture.go, internal/tools/browser/memory_bench_test.go.
package browser

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"io"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func parseResponse(body []byte) (response, error) {
	var result response
	invalid := func() (response, error) {
		return response{}, fault.Errorf(diag.BrowserFailed, diag.Text(diag.BrowserInvalidResponse))
	}
	decoder := jsontext.NewDecoder(bytes.NewBuffer(body))
	token, err := decoder.ReadToken()
	if err != nil || token.Kind() != '{' {
		return invalid()
	}
	start, end := 0, 0
	for decoder.PeekKind() != '}' {
		key, err := decoder.ReadToken()
		if err != nil || key.Kind() != '"' {
			return invalid()
		}
		name := key.String()
		value, err := decoder.ReadValue()
		if err != nil {
			return invalid()
		}
		switch name {
		case "data":
			end = int(decoder.InputOffset())
			start = end - len(value)
		case "success":
			if err := json.Unmarshal(value, &result.Success); err != nil || string(value) == "null" {
				return invalid()
			}
		case "error":
			if err := json.Unmarshal(value, &result.Error); err != nil {
				return invalid()
			}
		}
	}
	if _, err := decoder.ReadToken(); err != nil {
		return invalid()
	}
	if _, err := decoder.ReadToken(); err != io.EOF {
		return invalid()
	}
	if result.Success && end == 0 {
		return invalid()
	}
	result.Data = body[start:end:end]
	return result, nil
}
