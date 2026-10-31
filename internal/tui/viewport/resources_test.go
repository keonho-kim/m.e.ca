// Purpose: Check display cache eviction and heap release over repeated transcript replacements.
// Pattern: Resource lifecycle regression test.
// Usage: go test ./internal/tui/viewport -run TestDisplayResources -v.
// Related: internal/tui/viewport/window.go, internal/tui/viewport/block_layout.go.
package viewport

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func TestDisplayResources(t *testing.T) {
	var view View
	var transcript Transcript
	runtime.GC()
	var before, loaded, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for cycle := range 12 {
		view = View{Follow: true}
		for n := range 300 {
			view.AppendBlock(Block{Text: fmt.Sprintf("## %d %d\n\n%s", cycle, n, strings.Repeat("**body** text.\n\n", 100)), Markdown: true})
		}
		transcript.Prepare(80, &view, nil)
		rows := transcript.Window(max(0, transcript.Count()-30), 30)
		if len(rows) != 30 {
			t.Fatal(len(rows))
		}
		retained := 0
		for i := range view.Blocks {
			if view.Blocks[i].layout.parsed != nil {
				retained++
			}
		}
		if retained > 2 {
			t.Fatalf("%d offscreen parsed blocks retained", retained)
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&loaded)
	runtime.KeepAlive(view)
	runtime.KeepAlive(transcript)
	view = View{}
	transcript = Transcript{}
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("heap before=%d loaded=%d released=%d", before.HeapAlloc, loaded.HeapAlloc, after.HeapAlloc)
	if after.HeapAlloc > before.HeapAlloc+16*1024*1024 {
		t.Fatal("released transcripts retained more than 16MiB")
	}
}
