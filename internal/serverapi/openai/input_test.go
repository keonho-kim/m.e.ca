// Purpose: Check strict bounded parsing and measure large compatible request admission costs.
// Pattern: Boundary regression tests and allocation benchmarks.
// Usage: go test ./internal/serverapi/openai.
// Related: internal/serverapi/openai/input.go, internal/serverapi/openai/messages.go.
package openai

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParserRejectsAmbiguousAndUnboundedInput(t *testing.T) {
	for _, body := range []string{
		`{"model":"meca","input":[{"role":"user","ro\u006ce":"user","content":"hello"}]}`,
		`{"model":"meca","input":[{"role":"user","content":[{"type":"input_text","text":"x","te\u0078t":"y"}]}]}`,
		`{"model":"meca","input":"hello"} {}`,
		`{"model":"meca","input":[{"role":"user","content":[` + strings.Repeat(`{"type":"input_text","text":"x"},`, 16384) + `{"type":"input_text","text":"x"}]}]}`,
	} {
		r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if _, err := parse(r, httptest.NewRecorder(), "response"); err == nil {
			t.Fatal("ambiguous or excessive input accepted")
		}
	}
}

func TestParserPreservesEscapedTextAndChecksCancellation(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"meca","input":[{"role":"user","content":[{"type":"input_text","text":"a\/b\n"},{"type":"input_text","text":"\uD55C\uAE00"}]}]}`))
	r.Header.Set("Content-Type", "application/json")
	p, err := parse(r, httptest.NewRecorder(), "response")
	if err != nil || len(p.input.Messages) != 1 || p.input.Messages[0].Text != "a/b\n한글" {
		t.Fatal(p, err)
	}
	ctx := t.Context()
	r = httptest.NewRequestWithContext(ctx, "POST", "/v1/responses", strings.NewReader(`{"model":"meca","input":"hello"}`))
	r.Header.Set("Content-Type", "application/json")
	// A pre-cancelled request must not allocate or seed a conversation.
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := parse(r.WithContext(cancelled), httptest.NewRecorder(), "response"); err == nil {
		t.Fatal("cancelled input accepted")
	}
}

func BenchmarkParseMessages(b *testing.B) {
	body := `{"model":"meca","messages":[` + strings.Repeat(`{"role":"user","content":"hello"},`, 4095) + `{"role":"user","content":"hello"}]}`
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if _, err := parse(r, httptest.NewRecorder(), "chat"); err != nil {
			b.Fatal(err)
		}
	}
}
