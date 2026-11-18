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
	decoder := jsontext.NewDecoder(bytes.NewBuffer(body))
	result, err := readEnvelope(decoder, body, "data")
	if err != nil {
		return response{}, err
	}
	if _, err := decoder.ReadToken(); err != io.EOF {
		return response{}, invalidResponse()
	}
	return result, nil
}

func parseReadBatch(body []byte) (response, string, error) {
	decoder := jsontext.NewDecoder(bytes.NewBuffer(body))
	token, err := decoder.ReadToken()
	if err != nil || token.Kind() != '[' {
		return response{}, "", invalidResponse()
	}
	first, err := readEnvelope(decoder, body, "result")
	if err != nil || first.Index == nil || *first.Index != 0 {
		return response{}, "", invalidResponse()
	}
	result := first
	var address string
	if first.Success {
		address, err = responseURL(first.Data)
		if err != nil {
			return response{}, "", err
		}
		result, err = readEnvelope(decoder, body, "result")
		if err != nil || result.Index == nil || *result.Index != 1 {
			return response{}, "", invalidResponse()
		}
	}
	if token, err := decoder.ReadToken(); err != nil || token.Kind() != ']' {
		return response{}, "", invalidResponse()
	}
	if _, err := decoder.ReadToken(); err != io.EOF {
		return response{}, "", invalidResponse()
	}
	return result, address, nil
}

func invalidResponse() error {
	return fault.Errorf(diag.BrowserFailed, diag.Text(diag.BrowserInvalidResponse))
}

func readEnvelope(decoder *jsontext.Decoder, body []byte, dataField string) (response, error) {
	var result response
	token, err := decoder.ReadToken()
	if err != nil || token.Kind() != '{' {
		return response{}, invalidResponse()
	}
	start, end := 0, 0
	checked := false
	for decoder.PeekKind() != '}' {
		key, err := decoder.ReadToken()
		if err != nil || key.Kind() != '"' {
			return response{}, invalidResponse()
		}
		name := key.String()
		value, err := decoder.ReadValue()
		if err != nil {
			return response{}, invalidResponse()
		}
		switch name {
		case dataField:
			end = int(decoder.InputOffset())
			start = end - len(value)
		case "success":
			if err := json.Unmarshal(value, &result.Success); err != nil || string(value) == "null" {
				return response{}, invalidResponse()
			}
			checked = true
		case "error":
			if err := json.Unmarshal(value, &result.Error); err != nil {
				return response{}, invalidResponse()
			}
		case "index":
			if err := json.Unmarshal(value, &result.Index); err != nil {
				return response{}, invalidResponse()
			}
		}
	}
	if _, err := decoder.ReadToken(); err != nil {
		return response{}, invalidResponse()
	}
	if !checked || result.Success && end == 0 {
		return response{}, invalidResponse()
	}
	result.Data = body[start:end:end]
	return result, nil
}

func responseURL(data []byte) (string, error) {
	var value struct{ URL string }
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	if value.URL == "" {
		return "", invalidResponse()
	}
	return value.URL, nil
}
