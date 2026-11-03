// Purpose: Measure the same changing Markdown through interleaved history and incremental team streams.
// Pattern: Representative terminal frame benchmark.
// Usage: go test ./internal/tui/viewport -run '^$' -bench BenchmarkTeamStreamFrame -benchmem.
// Related: internal/tui/viewport/streams.go, internal/tui/viewport/update.go.
package viewport

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkTeamStreamFrame(b *testing.B) {
	prefix := strings.Repeat("- **Result**: preserved `value`.\n", 1000)
	values := []string{prefix + "one", prefix + "two"}
	for _, incremental := range []bool{false, true} {
		b.Run(fmt.Sprintf("incremental%t", incremental), func(b *testing.B) {
			view := View{Follow: true}
			view.AppendBlock(Block{Text: values[0], Markdown: true, Streaming: incremental})
			id := view.Next
			view.AppendBlock(Block{Text: "A concurrent peer is working.", Markdown: true})
			var layout Transcript
			layout.Prepare(80, &view, nil)
			layout.Window(max(0, layout.Count()-25), 25)
			step := 0
			b.ReportAllocs()
			for b.Loop() {
				view.Update(id, Block{Text: values[step%2], Markdown: true, Streaming: incremental})
				layout.Prepare(80, &view, nil)
				layout.Window(max(0, layout.Count()-25), 25)
				step++
			}
		})
	}
}
