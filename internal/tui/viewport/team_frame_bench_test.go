// Purpose: Measure changed and repeated team frames with a representative visible row budget.
// Pattern: Presentation allocation benchmark.
// Usage: go test ./internal/tui/viewport -run '^$' -bench 'Benchmark(ManyTeamCards|UnchangedTeamFrame)' -benchmem.
// Related: internal/tui/viewport/update.go, internal/tui/viewport/window.go, internal/tui/viewport/block_window.go.
package viewport

import "testing"

func BenchmarkManyTeamCards(b *testing.B) {
	view := View{Follow: true}
	for range 20 {
		view.AppendBlock(Block{Text: "**Hana**", Markdown: true, KeepWithNext: true, SpaceAfter: 1})
		view.AppendBlock(Block{Text: "first body", Markdown: true, SpaceAfter: 2})
	}
	id := view.Next
	values := []string{"first body", "second body"}
	var layout Transcript
	layout.Prepare(80, &view, nil)
	layout.Window(layout.Count()-50, 50)
	step := 0
	b.ReportAllocs()
	for b.Loop() {
		view.Update(id, Block{Text: values[step%2], Markdown: true, Streaming: true, SpaceAfter: 2})
		layout.Prepare(80, &view, nil)
		layout.Window(layout.Count()-50, 50)
		step++
	}
}

func BenchmarkUnchangedTeamFrame(b *testing.B) {
	for _, update := range []bool{false, true} {
		name := "cached"
		if update {
			name = "identical_update"
		}
		b.Run(name, func(b *testing.B) {
			view := View{Follow: true}
			value := Block{Text: "current output", Markdown: true, Streaming: true, SpaceAfter: 2}
			for range 20 {
				view.AppendBlock(Block{Text: "**Hana**", Markdown: true, KeepWithNext: true, SpaceAfter: 1})
				view.AppendBlock(Block{Text: "completed output", Markdown: true, SpaceAfter: 2})
			}
			id := view.Next
			view.Update(id, value)
			var layout Transcript
			layout.Prepare(80, &view, nil)
			layout.Window(layout.Count()-50, 50)
			b.ReportAllocs()
			for b.Loop() {
				if update {
					view.Update(id, value)
				}
				layout.Prepare(80, &view, nil)
				layout.Window(layout.Count()-50, 50)
			}
		})
	}
}
