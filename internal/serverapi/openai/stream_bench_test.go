// Purpose: Measure wire encoding and framing without network or recorder accumulation.
// Pattern: Allocation benchmark at the streaming adapter boundary.
// Usage: go test ./internal/serverapi/openai -run '^$' -bench BenchmarkStream -benchmem.
// Related: internal/serverapi/openai/stream.go, internal/serverapi/eventstream/writer.go.
package openai

import (
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	"net/http"
	"strings"
	"testing"
)

type discardStream struct {
	header                 http.Header
	bytes, writes, flushes int
}

func (w *discardStream) Header() http.Header { return w.header }
func (w *discardStream) WriteHeader(int)     {}
func (w *discardStream) Write(p []byte) (int, error) {
	w.bytes += len(p)
	w.writes++
	return len(p), nil
}
func (w *discardStream) Flush() { w.flushes++ }

func BenchmarkStream(b *testing.B) {
	for _, chat := range []bool{true, false} {
		name := "responses"
		if chat {
			name = "chat"
		}
		b.Run(name, func(b *testing.B) {
			out := &discardStream{header: make(http.Header)}
			w, err := eventstream.New(out)
			if err != nil {
				b.Fatal(err)
			}
			e := encoder{writer: w, chat: chat, value: server.Response{ID: "resp_" + strings.Repeat("a", 64), Created: 1}}
			if err := e.start(); err != nil {
				b.Fatal(err)
			}
			text := strings.Repeat("x", 64)
			b.ReportAllocs()
			for b.Loop() {
				if err := e.delta(text); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	b.Run("terminal_1MiB", func(b *testing.B) {
		out := &discardStream{header: make(http.Header)}
		w, err := eventstream.New(out)
		if err != nil {
			b.Fatal(err)
		}
		value := server.Response{ID: "resp_" + strings.Repeat("a", 64), Created: 1, Status: "completed"}
		e := encoder{writer: w, value: value, started: true, messageID: messageID(value)}
		text := strings.Repeat("x", 1<<20)
		b.ReportAllocs()
		for b.Loop() {
			if err := e.finish(value, text); err != nil {
				b.Fatal(err)
			}
		}
	})
}
