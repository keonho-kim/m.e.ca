// Purpose: Check bounded draft retention and measure compatible text projection allocations.
// Pattern: Focused projection regression test and benchmark.
// Usage: go test ./internal/serverapi/openai.
// Related: internal/serverapi/openai/output.go.
package openai

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
)

func TestClippedCommitReleasesPendingWhitespace(t *testing.T) {
	p := newOutput(false, func(string) error { return nil })
	if err := p.event(server.Event{Record: record.Event{Kind: "assistant_delta", Text: "body" + strings.Repeat(" ", 1<<20)}}); err != nil {
		t.Fatal(err)
	}
	if err := p.event(server.Event{Truncated: true, Record: record.Event{Kind: "response"}}); err != nil {
		t.Fatal(err)
	}
	if !p.buffered || p.pending.Cap() != 0 {
		t.Fatal("buffered projection retained an unused draft", p.pending.Cap())
	}
}

func TestWhitespaceBufferIsBoundedAndFinalOutputPreserved(t *testing.T) {
	var output strings.Builder
	p := newOutput(false, func(text string) error { output.WriteString(text); return nil })
	text := "body" + strings.Repeat(" ", 128<<10)
	if err := p.event(server.Event{Record: record.Event{Kind: "assistant_delta", Text: text}}); err != nil {
		t.Fatal(err)
	}
	if !p.buffered || p.pending.Cap() != 0 {
		t.Fatal("large whitespace retained for every observer")
	}
	final := text + "end"
	if err := p.finish(final); err != nil || output.String() != final {
		t.Fatal("bounded buffering changed final output", err)
	}
}

func BenchmarkOutputAppend(b *testing.B) {
	p := newOutput(false, func(string) error { return nil })
	text := strings.Repeat("x", 64)
	b.ReportAllocs()
	for b.Loop() {
		if err := p.append(text); err != nil {
			b.Fatal(err)
		}
	}
}
