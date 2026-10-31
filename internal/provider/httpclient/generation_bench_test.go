// Purpose: Measure complete local HTTP generation with and without execution-scoped request reuse.
// Pattern: End-to-end adapter benchmark against a local streaming fixture.
// Usage: Run BenchmarkGenerationEncoding with -benchmem; this excludes real model inference.
// Related: internal/provider/httpclient/run_generator.go, internal/provider/protocol/growth_bench_test.go.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func BenchmarkGenerationEncoding(b *testing.B) {
	for _, size := range []int{2048, 1 << 20, 8 << 20} {
		for _, cached := range []bool{false, true} {
			b.Run(fmt.Sprintf("bytes%d/cache=%v", size, cached), func(b *testing.B) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if _, err := io.Copy(io.Discard, r.Body); err != nil {
						b.Error(err)
					}
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"answer\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
				}))
				defer server.Close()
				conn, err := connection.New(connection.OpenRouter, server.URL)
				if err != nil {
					b.Fatal(err)
				}
				client, err := NewClient(conn, "")
				if err != nil {
					b.Fatal(err)
				}
				defer client.Close()
				var generator modelapi.Generator = client
				if cached {
					var release func()
					generator, release = client.NewRunGenerator()
					defer release()
				}
				messages := make([]modelapi.Message, 41)
				for i := range 40 {
					messages[i] = modelapi.Message{Role: "user", Content: strings.Repeat("x", max(1, size/40))}
				}
				request := modelapi.Request{ModelID: "model", MaxOutput: 123, Messages: messages}
				step := 0
				b.ReportAllocs()
				for b.Loop() {
					request.Messages[40] = modelapi.Message{Role: "user", Content: []string{"next 0", "next 1"}[step%2]}
					if err := generator.Generate(context.Background(), request, func(modelapi.Event) error { return nil }); err != nil {
						b.Fatal(err)
					}
					step++
				}
			})
		}
	}
}
