// Purpose: Verify sparse preview layout returns numbered visible rows and preserves scrolling bounds.
// Pattern: Pure presentation contract tests and benchmark.
// Usage: go test ./internal/tui/browser.
// Related: internal/tui/browser/preview.go, internal/tui/session/file_preview.go.
package browser

import (
	"strings"
	"testing"
)

func TestPreviewVisibleWindow(t *testing.T) {
	p := Preview{Content: "hello\nworld\n"}
	if p.Count(80) != 2 {
		t.Fatal("wrong row count")
	}
	rows := p.Window(80, 1, 1)
	if len(rows) != 1 || rows[0] != "  2 | world" {
		t.Fatal(rows)
	}
	p.Content = strings.Repeat("x\n", 131072)
	if p.Count(80) != 131072 {
		t.Fatal("large preview row count changed")
	}
	rows = p.Window(80, 131070, 10)
	if len(rows) != 2 || !strings.Contains(rows[1], "131072 | x") {
		t.Fatal(rows)
	}
	p.Truncated = true
	if p.Count(80) != 131073 || len(p.Window(80, 131072, 5)) != 1 {
		t.Fatal("truncation marker lost")
	}
}

func BenchmarkPreviewWindow(b *testing.B) {
	text := strings.Repeat("x\n", 131072)
	b.ReportAllocs()
	for b.Loop() {
		p := Preview{Content: text}
		p.Window(80, 131050, 24)
	}
}
