// Purpose: Measure live transcript updates as immutable terminal history grows.
// Pattern: Representative frame workload benchmark.
// Usage: Run BenchmarkLiveTranscriptFrame with -benchmem before and after index changes.
// Related: internal/tui/viewport/window.go, internal/tui/session/chat_projection.go.
package viewport

import (
	"fmt"
	"testing"
)

func BenchmarkLiveTranscriptFrame(b *testing.B) {
	for _, size := range []int{1, 200, 1000} {
		b.Run(fmt.Sprintf("history%d", size), func(b *testing.B) {
			view := View{Follow: true}
			for i := range size {
				view.Append(fmt.Sprintf("Observed operation %d\n  preserved result\n", i), false)
			}
			live := Block{Text: "working 0", ID: view.Next + 1}
			var transcript Transcript
			transcript.Prepare(80, &view, []*Block{&live})
			transcript.Window(max(0, transcript.Count()-30), 30)
			values := []string{"working 0", "working 1"}
			step := 0
			b.ReportAllocs()
			for b.Loop() {
				live.Text = values[step%2]
				transcript.Prepare(80, &view, []*Block{&live})
				rows := transcript.Window(max(0, transcript.Count()-30), 30)
				if len(rows) == 0 || rows[len(rows)-1].Text != live.Text {
					b.Fatal("live update missing")
				}
				step++
			}
		})
	}
}
