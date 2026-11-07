// Purpose: Reject oversized single-line and multiline streams for every provider adapter.
// Pattern: Controllable stream input boundary tests.
// Usage: go test ./internal/provider/protocol -run TestAllProviderFrameLimits.
// Related: internal/provider/protocol/stream.go, internal/provider/protocol/lines.go.
package protocol

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestAllProviderFrameLimits(t *testing.T) {
	const limit = 8192
	for _, provider := range connection.Providers() {
		for _, multiline := range []bool{false, true} {
			t.Run(string(provider.Kind)+map[bool]string{false: "/line", true: "/multiline"}[multiline], func(t *testing.T) {
				wire := strings.Repeat("x", limit+1)
				if multiline {
					wire = strings.Repeat("data: "+strings.Repeat("x", 2048)+"\n", 9)
				}
				if provider.Kind == connection.Ollama && multiline {
					return // Ollama is NDJSON and has no multiline SSE envelope.
				}
				response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(wire))}
				var truncated *modelapi.OutputLimitError
				err := readResponse(context.Background(), response, provider.Kind, limit, func(modelapi.Event) error { return nil })
				if !errors.As(err, &truncated) {
					t.Fatal("frame budget not enforced before decoding", err)
				}
			})
		}
	}
}
