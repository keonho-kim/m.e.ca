// Purpose: Keep newline-dense code readable without allocating complete logical row arrays.
// Pattern: Resource-bound display regression test and cold benchmark.
// Usage: go test ./internal/tui/viewport -run TestDense -bench BenchmarkDenseMarkdown.
// Related: internal/tui/viewport/markdown_walk.go, internal/tui/markdown/render_packed.go.
package viewport

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestDenseMarkdownBoundsTransientRows(t *testing.T) {
	text := "```\n" + strings.Repeat("\n", 256<<10) + "```"
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	block := Block{Text: text, Markdown: true}
	count := block.Count(80)
	rows := block.Window(80, count-2, 2)
	runtime.ReadMemStats(&after)
	if count != 256<<10 || len(rows) != 2 || rows[0].Text != "│ " || block.Text != text {
		t.Fatal("code rows or original changed", count)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 64<<20 {
		t.Fatalf("dense source allocated %d bytes", allocated)
	}
}

func BenchmarkDenseMarkdown(b *testing.B) {
	text := "```\n" + strings.Repeat("\n", 1<<20) + "```"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block := Block{Text: text, Markdown: true}
		count := block.Count(80)
		if len(block.Window(80, count-30, 30)) != 30 {
			b.Fatal("lost rows")
		}
	}
}

func TestSparseMarkdownWindowsPreserveLargeOriginal(t *testing.T) {
	var source strings.Builder
	source.WriteString("```\n")
	for i := range 500000 {
		source.WriteString(strconv.Itoa(i))
		source.WriteByte('\n')
	}
	source.WriteString("```")
	text := source.String()
	block := Block{Text: text, Markdown: true}
	count := block.Count(80)
	if !block.layout.sparse || count != 500000 {
		t.Fatal("large document did not select bounded sparse rows", count)
	}
	last := block.Window(80, count-1, 1)
	first := block.Window(80, 0, 1)
	if len(last) != 1 || last[0].Text != "│ 499999" || len(first) != 1 || first[0].Text != "│ 0" {
		t.Fatal("sparse scrolling changed code contents")
	}
	if block.layout.parsed != nil || block.layout.packed != nil || block.Text != text {
		t.Fatal("sparse window retained complete rows or changed the original")
	}
}
