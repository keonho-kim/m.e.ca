// Purpose: Reject malformed native cloud frames without panics or executable partial calls.
// Pattern: Bounded provider stream fuzz test.
// Usage: go test ./internal/provider/protocol -run '^$' -fuzz FuzzCloudResponse -fuzztime=5s.
// Related: internal/provider/protocol/stream.go, internal/provider/protocol/chunk.go.
package protocol

import (
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func FuzzCloudResponse(f *testing.F) {
	for kind, body := range []string{`{"status":"completed","output":[]}`, `{"type":"message","content":[],"stop_reason":"end_turn"}`, `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"answer"}]}}]}`} {
		f.Add(uint8(kind), body, false)
		f.Add(uint8(kind), "data: "+body+"\n\n", true)
	}
	f.Fuzz(func(t *testing.T, index uint8, body string, sse bool) {
		if len(body) > 64<<10 {
			return
		}
		kind := []connection.ProviderKind{connection.OpenAI, connection.Claude, connection.Gemini}[int(index)%3]
		response := &http.Response{Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
		response.Header.Set("Content-Type", "application/json")
		if sse {
			response.Header.Set("Content-Type", "text/event-stream")
		}
		done := false
		err := ReadResponse(t.Context(), response, kind, func(event modelapi.Event) error {
			if len(event.Calls) > 0 && !event.Done {
				t.Fatal("partial function call exposed")
			}
			done = done || event.Done
			return nil
		})
		if err != nil && done {
			t.Fatal("failed native response was already completed")
		}
	})
}
