// Purpose: Preserve rendered rows and bounded ownership when a Markdown stream finishes.
// Pattern: Presentation lifecycle regression tests and benchmark.
// Usage: Viewport tests compare final windows with freshly rendered source.
// Related: internal/tui/viewport/streams.go, internal/tui/viewport/update.go.
package viewport

import (
	"reflect"
	"strings"
	"testing"
)

func TestMarkdownCompletionKeepsPreparedRows(t *testing.T) {
	for _, update := range []bool{false, true} {
		v := View{Follow: true}
		text := "## Report\n\n| Name | Value |\n|---|---|\n| long content | **한글** |\n\n- a\n- b\n\n```go\ncode\n```\n"
		v.AppendBlock(Block{Text: text, Markdown: true, Streaming: true})
		var transcript Transcript
		transcript.Prepare(30, &v, nil)
		transcript.Window(0, transcript.Count())
		if update {
			v.Update(v.Next, Block{Text: text, Markdown: true})
		} else {
			v.FinishStream(v.Next)
		}
		if len(v.Blocks[0].layout.parsed) == 0 && v.Blocks[0].layout.packed == nil {
			t.Fatal("completion discarded reusable rendered rows")
		}
		if v.Blocks[0].stream.RetainedBytes() != 0 || v.Blocks[0].layout.metrics != nil {
			t.Fatal("completion retained live parsing state")
		}
		transcript.Prepare(30, &v, nil)
		fresh := Block{Text: text, Markdown: true, ID: v.Next}
		got, want := transcript.Window(0, transcript.Count()), fresh.Window(30, 0, fresh.Count(30))
		if !reflect.DeepEqual(got, want) {
			t.Fatal("completion changed visible content")
		}
	}
}

func BenchmarkMarkdownCompletion(b *testing.B) {
	text := strings.Repeat("- **Result**: observation.\n", 2000)
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		v := View{Follow: true}
		v.AppendBlock(Block{Text: text, Markdown: true, Streaming: true})
		var transcript Transcript
		transcript.Prepare(80, &v, nil)
		transcript.Window(0, 25)
		b.StartTimer()
		v.FinishStream(v.Next)
		transcript.Prepare(80, &v, nil)
		transcript.Window(25, 25)
	}
}

func TestMarkdownCompletionReleasesOffscreenAndChangedRows(t *testing.T) {
	for _, changed := range []bool{false, true} {
		v := View{Follow: true}
		v.AppendBlock(Block{Text: "**original**\n\ntext", Markdown: true, Streaming: true})
		var transcript Transcript
		transcript.Prepare(30, &v, nil)
		if changed {
			transcript.Window(0, transcript.Count())
			v.Update(v.Next, Block{Text: "replacement", Markdown: true})
		} else {
			v.FinishStream(v.Next)
		}
		if v.Blocks[0].cacheBytes() != 0 {
			t.Fatal("offscreen or stale streaming rows retained")
		}
		transcript.Prepare(30, &v, nil)
		if rows := transcript.Window(0, transcript.Count()); len(rows) == 0 {
			t.Fatal("released source cannot be rendered")
		}
	}
}
