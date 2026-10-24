// Purpose: Check streamed Markdown against complete parsing, including syntax changed by later text.
// Pattern: Differential presentation tests and benchmark.
// Usage: go test ./internal/tui/markdown -run TestStream -bench BenchmarkStream.
// Related: internal/tui/markdown/stream.go, internal/tui/markdown/render.go.
package markdown

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestStreamMatchesCompleteRendering(t *testing.T) {
	for _, source := range []string{
		"# Title\n\nFirst **paragraph**.\n\nSecond paragraph.\n\nThird.\n\n끝 😀\n",
		"- one\n\n- two\n  continued\n\n  paragraph\n\nend\n",
		"before\n\n```go\nx := 1\n\n```\n\nafter\n\nlast",
		"title\n=====\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nnext",
		"[later][id]\n\nother\n\n[id]: https://example.com\n",
		"> quote\n>\n> more\n\n---\n\nlast\n\n",
	} {
		var stream Stream
		for at := range source {
			text := source[:at]
			if got, want := stream.Render(text), Render(text); !equalLines(got, want) {
				t.Fatalf("prefix %q:\ngot %#v\nwant %#v", text, got, want)
			}
		}
		if got := stream.Render(source); !equalLines(got, Render(source)) {
			t.Fatalf("complete %q: %#v", source, got)
		}
		for _, replacement := range []string{source + "new", "reset", ""} {
			if got := stream.Render(replacement); !equalLines(got, Render(replacement)) {
				t.Fatalf("replacement %q: %#v", replacement, got)
			}
		}
	}
}

func BenchmarkStreamTail(b *testing.B) {
	text := strings.Repeat("**Result**: example and `code`.\n\n", 4000)
	var stream Stream
	stream.Render(text)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		stream.Render(text + strings.Repeat("x", i%100))
		i++
	}
}

func BenchmarkFullMarkdown(b *testing.B) {
	text := strings.Repeat("**Result**: example and `code`.\n\n", 4000)
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		Render(text + strings.Repeat("x", i%100))
		i++
	}
}

func FuzzStream(f *testing.F) {
	for _, text := range []string{"#", "0)", "a\n\nb\n\nc\n\n", "- a\n\n- b\n\nend", "```\ncode\n```\n\nend", "[a][b]\n\n[b]: https://example.com"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 8192 {
			t.Skip()
		}
		var stream Stream
		for at := 0; at <= len(text); at += max(1, len(text)/32) {
			got, want := stream.Render(text[:at]), Render(text[:at])
			if !equalLines(got, want) {
				t.Fatalf("prefix %q: %#v want %#v", text[:at], got, want)
			}
		}
	})
}

func equalLines(a, b []Line) bool {
	return slices.EqualFunc(a, b, func(a, b Line) bool { return reflect.DeepEqual(a, b) })
}
