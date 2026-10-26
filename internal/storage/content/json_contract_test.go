// Purpose: Preserve stored JSON bytes, numeric identities and independent decoded records.
// Pattern: Persistence serialization contract tests.
// Usage: Run TestStoredJSON and FuzzStoredJSON when changing the storage codec.
// Related: internal/storage/content/pack.go, internal/storage/sqlite/messages.go.
package content

import (
	"encoding/json"
	"reflect"
	"testing"

	fastjson "github.com/goccy/go-json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestStoredJSONEncoding(t *testing.T) {
	for _, value := range []any{
		map[string]any{"z": "<module>&\u2028\u2029한국어", "a": json.RawMessage(`{ "n":9007199254740993, "text":"a\nb" }`)},
		modelapi.Message{Role: "assistant", Content: "\xff", Calls: []toolapi.Call{{ID: "call_1", Name: "execute_command", Arguments: json.RawMessage(`{"command":"echo 한국어","number":9007199254740993}`)}}},
		map[string]any{"nil": []string(nil), "empty": []string{}},
		map[string]any{"raw": json.RawMessage(`{invalid}`)},
	} {
		want, expected := json.Marshal(value)
		got, actual := fastjson.Marshal(value)
		if (actual == nil) != (expected == nil) || actual == nil && string(got) != string(want) {
			t.Fatalf("stored encoding differs: %s (%v), want %s (%v)", got, actual, want, expected)
		}
	}
}

func compareStoredJSON(t *testing.T, body []byte) {
	t.Helper()
	var got, want modelapi.Message
	expected := json.Unmarshal(body, &want)
	actual := fastjson.Unmarshal(body, &got)
	if (actual == nil) != (expected == nil) {
		t.Fatalf("decode error %v; standard error %v", actual, expected)
	}
	if actual == nil && !reflect.DeepEqual(got, want) {
		t.Fatalf("stored message differs: %#v, want %#v", got, want)
	}
}

func TestStoredJSONDecoding(t *testing.T) {
	for _, body := range []string{
		`{"role":"assistant","content":"한국어\n<module>&","calls":[{"id":"call_1","name":"execute_command","arguments":{ "command":"echo a", "n":9007199254740993 }}]}`,
		`{"role":"user","content":"\ud800","unknown":[true,null]}`,
		"{\"content\":\"\xff\"}",
		`{"content":123}`,
		`{"content":"first","content":null}`,
		`{"content":"valid"} {}`,
		"{\"content\":\"unescaped\nnewline\"}",
		`{"unknown":{"nested":INVALID}}`,
	} {
		t.Run(body, func(t *testing.T) { compareStoredJSON(t, []byte(body)) })
	}
}

func TestStoredJSONOwnsDecodedValues(t *testing.T) {
	body := []byte(`{"content":"independent","calls":[{"id":"call_1","name":"execute_command","arguments":{ "command": "echo a" }}]}`)
	var got modelapi.Message
	if err := fastjson.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	for i := range body {
		body[i] = 'x'
	}
	if got.Content != "independent" || len(got.Calls) != 1 || string(got.Calls[0].Arguments) != `{ "command": "echo a" }` {
		t.Fatalf("decoded record aliases its input: %#v", got)
	}
}

func FuzzStoredJSON(f *testing.F) {
	for _, body := range []string{`{"content":"hello"}`, `{"calls":[]}`, `{"unknown":[true,null]}`} {
		f.Add([]byte(body))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > 64*1024 {
			return
		}
		compareStoredJSON(t, body)
	})
}
