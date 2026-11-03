// Purpose: Verify interleaved stream replacement preserves history order and sparse row layout.
// Pattern: Incremental-versus-fresh projection regression test.
// Usage: go test ./internal/tui/viewport -run TestStableBlock.
// Related: internal/tui/viewport/update.go, internal/tui/viewport/transcript_index.go.
package viewport

import (
	"reflect"
	"testing"
)

func TestStableBlockUpdateMatchesFreshLayout(t *testing.T) {
	v := View{Follow: true}
	v.AppendBlock(Block{Text: "earlier history\n\n", Markdown: true})
	v.AppendBlock(Block{Text: "**Hana**\n\nfirst", Markdown: true})
	a := v.Next
	v.AppendBlock(Block{Text: "**Ren**\n\nsecond", Markdown: true})
	var cached Transcript
	cached.Prepare(35, &v, nil)
	for _, text := range []string{"first update\n\nmore text", "", "a completed answer with a longer paragraph and wrapped lines"} {
		v.Update(a, Block{Text: text, Markdown: true})
		cached.Prepare(35, &v, nil)
		var fresh Transcript
		fresh.Prepare(35, &v, nil)
		got, want := cached.Window(0, cached.Count()), fresh.Window(0, fresh.Count())
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cached layout differs after %q", text)
		}
	}
}
