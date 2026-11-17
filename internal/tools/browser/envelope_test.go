// Purpose: Verify zero-copy response ranges and strict helper envelope validation.
// Pattern: Parser contract regression with malformed and ambiguous inputs.
// Usage: go test ./internal/tools/browser -run TestResponseEnvelope.
// Related: internal/tools/browser/envelope.go.
package browser

import (
	"bytes"
	"testing"
)

func TestResponseEnvelope(t *testing.T) {
	for _, body := range []string{`{"success":true,"data":{"text":"hello"}}`, ` { "data" : {"text":"hello"}, "extra":[1,2], "success":true } `} {
		raw := []byte(body)
		value, err := parseResponse(raw)
		if err != nil || string(value.Data) != `{"text":"hello"}` {
			t.Fatal(value, err)
		}
		offset := bytes.Index(raw, value.Data)
		if offset < 0 || &raw[offset] != &value.Data[0] {
			t.Fatal("copied envelope data")
		}
	}
	for _, body := range []string{`[]`, `{"success":true}`, `{"success":true,"data":{}} {}`, `{"success":true,"success":false,"data":{}}`, `{"success":true,"data":{},"data":{}}`, `{"success":null,"data":{}}`, `{"success":true,"data":{"text":"unterminated}}`} {
		if _, err := parseResponse([]byte(body)); err == nil {
			t.Fatal("accepted invalid envelope", body)
		}
	}
}
