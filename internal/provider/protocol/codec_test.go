// Purpose: Preserve wire JSON values, deterministic map ordering and raw-value validation.
// Pattern: Serialization contract tests.
// Usage: Run TestWireJSONEncode and FuzzWireJSONEncode for the provider encoder.
// Related: internal/provider/protocol/codec.go, internal/provider/protocol/request.go.
package protocol

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func compareWireEncoding(t *testing.T, value any) {
	t.Helper()
	expected, expectedErr := json.Marshal(value)
	actual, actualErr := wireJSON.Marshal(value)
	if (actualErr == nil) != (expectedErr == nil) {
		t.Fatalf("encode error %v; standard error %v", actualErr, expectedErr)
	}
	if actualErr != nil {
		return
	}
	// Compare decoded values: JSON HTTP does not require HTML escapes.
	parse := func(body []byte) any {
		decoder := json.NewDecoder(strings.NewReader(string(body)))
		decoder.UseNumber()
		var decoded any
		if err := decoder.Decode(&decoded); err != nil {
			t.Fatal(err)
		}
		return decoded
	}
	if !reflect.DeepEqual(parse(actual), parse(expected)) {
		t.Fatalf("different JSON values: %s / %s", actual, expected)
	}
}

func TestWireJSONEncode(t *testing.T) {
	value := map[string]any{
		"z": "<module>&\u2028한국어", "a": json.RawMessage(`{ "n":9007199254740993,"arguments":"a\nb" }`),
		"nil": []string(nil), "empty": []string{},
	}
	compareWireEncoding(t, value)
	first, err := wireJSON.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := wireJSON.Marshal(value)
	if err != nil || string(first) != string(second) || !strings.HasPrefix(string(first), `{"a":`) {
		t.Fatalf("nondeterministic encoding: %s / %s: %v", first, second, err)
	}
	compareWireEncoding(t, map[string]any{"raw": json.RawMessage(`{invalid}`)})
}

func FuzzWireJSONEncode(f *testing.F) {
	for _, text := range []string{"hello", "한국어\n<module>&", "\xff", "\u2028\u2029", strings.Repeat("x", 4096)} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 64*1024 {
			return
		}
		compareWireEncoding(t, map[string]any{"content": text})
	})
}
