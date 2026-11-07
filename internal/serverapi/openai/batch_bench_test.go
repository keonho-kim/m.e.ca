// Purpose: Compare identical SSE delta bursts and their actual HTTP flush counts.
// Pattern: Warm transport encoding benchmark with a non-retaining response sink.
// Usage: go test ./internal/serverapi/openai -run '^$' -bench BenchmarkStreamBatch -benchmem.
// Related: internal/serverapi/openai/stream_encoder.go, internal/serverapi/eventstream/batch.go.
package openai

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
)

func BenchmarkStreamBatch(b *testing.B) {
	for _, chat := range []bool{false, true} {
		for _, batch := range []bool{false, true} {
			b.Run(fmt.Sprintf("chat=%v/batch=%v", chat, batch), func(b *testing.B) {
				out := &discardStream{header: make(http.Header)}
				writer, err := eventstream.New(out)
				if err != nil {
					b.Fatal(err)
				}
				e := encoder{writer: writer, chat: chat, value: server.Response{ID: "resp_" + strings.Repeat("a", 64), Created: 1}}
				if err := e.start(); err != nil {
					b.Fatal(err)
				}
				if err := e.delta("initial"); err != nil {
					b.Fatal(err)
				}
				text := strings.Repeat("x", 64)
				send := func() error {
					for range 32 {
						if err := e.delta(text); err != nil {
							return err
						}
					}
					return nil
				}
				before := out.flushes
				b.ReportAllocs()
				for b.Loop() {
					if batch {
						err = writer.Batch(send)
					} else {
						err = send()
					}
					if err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(out.flushes-before)/float64(b.N), "flushes/op")
			})
		}
	}
}
