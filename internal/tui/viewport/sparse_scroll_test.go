// Purpose: Bound repeated sparse Markdown windows without reparsing complete large code documents.
// Pattern: Observable allocation regression test and retained-layout benchmark.
// Usage: go test ./internal/tui/viewport -run TestSparseScroll -bench BenchmarkSparseMarkdownScroll.
// Related: internal/tui/viewport/markdown_walk.go, internal/tui/markdown/plan.go.
package viewport

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func sparseCodeSource() string {
	var source strings.Builder
	source.WriteString("```\n")
	for index := range 500000 {
		source.WriteString(strconv.Itoa(index))
		source.WriteByte('\n')
	}
	source.WriteString("```")
	return source.String()
}

func TestSparseScrollDoesNotReparseTheWholeCodeBlock(t *testing.T) {
	block := Block{Text: sparseCodeSource(), Markdown: true}
	count := block.Count(80)
	if !block.layout.sparse {
		t.Fatal("fixture did not require sparse rows")
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	rows := block.Window(80, count-30, 30)
	runtime.ReadMemStats(&after)
	if len(rows) != 30 || rows[29].Text != "│ 499999" {
		t.Fatal("sparse window changed")
	}
	if bytes := after.TotalAlloc - before.TotalAlloc; bytes > 2<<20 {
		t.Fatalf("30-row sparse window allocated %d bytes", bytes)
	}
	block.releaseParsed()
	if block.layout.plan != nil || block.cacheBytes() != 0 {
		t.Fatal("inactive block retained its plan")
	}
	if rows := block.Window(80, 0, 1); len(rows) != 1 || rows[0].Text != "│ 0" || block.layout.plan == nil {
		t.Fatal("reactivated block did not rebuild its bounded plan")
	}
	if rows := block.Window(10, count-1, 1); len(rows) != 1 || rows[0].Text != "│ 499999" {
		t.Fatal("resize changed code content")
	}
}

func BenchmarkSparseMarkdownScroll(b *testing.B) {
	block := Block{Text: sparseCodeSource(), Markdown: true}
	count := block.Count(80)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if len(block.Window(80, count-30, 30)) != 30 {
			b.Fatal("lost rows")
		}
	}
}
