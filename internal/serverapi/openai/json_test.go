// Purpose: Verify validated JSON slicing and terminal text reuse preserve exact public values.
// Pattern: Wire-format regression and fuzz tests.
// Usage: go test ./internal/serverapi/openai; go test -fuzz FuzzParseJSON ./internal/serverapi/openai.
// Related: internal/serverapi/openai/json.go, internal/serverapi/openai/stream_text.go.
package openai

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
)

func TestTerminalEventsShareOriginalEscapedText(t *testing.T) {
	text := strings.Repeat("\"text\":null <&> \\ 한글\n", 10000)
	r := httptest.NewRecorder()
	w, err := eventstream.New(r)
	if err != nil {
		t.Fatal(err)
	}
	value := server.Response{ID: "resp_" + strings.Repeat("a", 64), Created: 1, Status: "completed"}
	e := encoder{writer: w, value: value}
	if err := e.start(); err != nil {
		t.Fatal(err)
	}
	if err := e.delta("prefix"); err != nil {
		t.Fatal(err)
	}
	if err := e.finish(value, text); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range strings.Split(r.Body.String(), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event struct {
			Type     string
			Text     string
			Part     outputText
			Item     messageValue
			Response responseObject
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			t.Fatal(err)
		}
		var content any
		switch event.Type {
		case "response.output_text.done":
			content = event.Text
		case "response.content_part.done":
			content = event.Part.Text
		case "response.output_item.done":
			content = event.Item.Content[0].Text
		case "response.completed":
			content = event.Response.Output[0].Content[0].Text
		default:
			continue
		}
		if content != text {
			t.Fatal("terminal event changed original text", event.Type)
		}
		count++
	}
	if count != 4 {
		t.Fatal("missing terminal events", count)
	}
}

func FuzzParseJSON(f *testing.F) {
	for _, seed := range []string{`{"model":"meca","input":"hello"}`, `{"model":"meca","input":[{"role":"user","content":[{"type":"input_text","text":"a\/b"}]}]}`, `{"model":"meca","model":"meca","input":"hello"}`, `{"":null}`, `[]`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		if len(body) > 1<<20 {
			return
		}
		r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		p, err := parse(r, httptest.NewRecorder(), "response")
		if err == nil && (!json.Valid([]byte(body)) || len(p.input.Messages) == 0) {
			t.Fatal("invalid document accepted")
		}
	})
}
