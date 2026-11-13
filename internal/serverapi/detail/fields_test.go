// Purpose: Preserve exact large tool fields and escaped text through bounded semantic projection.
// Pattern: Value and masking contracts over public detail projections.
// Usage: go test ./internal/serverapi/detail -run 'TestToolFields|TestJSONText'.
// Related: internal/serverapi/detail/fields.go, internal/serverapi/detail/json_string.go.
package detail

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestToolFieldsSelectWithoutCopyingLargeValues(t *testing.T) {
	raw := json.RawMessage(`{"stdout":"` + strings.Repeat("text", 10000) + `","meta":{"value":[1,true,null]},"empty":""}`)
	fields, err := selectToolField(t.Context(), raw, "", true)
	if err != nil || !reflect.DeepEqual(fields, []string{"empty", "meta", "stdout"}) {
		t.Fatal(fields, err)
	}
	value, err := selectToolField(t.Context(), raw, "meta", false)
	if err != nil || string(value.(json.RawMessage)) != `{"value":[1,true,null]}` {
		t.Fatal(value, err)
	}
	selected, err := selectToolField(t.Context(), raw, "stdout", false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, selected.(json.RawMessage)) {
		t.Fatal("field changed")
	}
	if _, err := selectToolField(t.Context(), raw, "missing", false); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestJSONTextKeepsEscapedSurrogatesAcrossChunks(t *testing.T) {
	for _, suffix := range []string{`\ud83d\ude00`, `\u0041\ud83d\ude00`, `\n\t\"\\`, `\ud800x`} {
		for offset := 8180; offset < 8200; offset++ {
			raw := []byte(`"` + strings.Repeat("x", offset) + suffix + `"`)
			var expected string
			if err := json.Unmarshal(raw, &expected); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err := writeJSONText(t.Context(), &out, raw, func() func(string, bool) string { return func(value string, _ bool) string { return value } })
			if err != nil || out.String() != expected {
				t.Fatal("string boundary changed", offset, err)
			}
		}
	}
}
