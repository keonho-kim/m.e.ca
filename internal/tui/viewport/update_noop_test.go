// Purpose: Verify unchanged display updates retain caches while real presentation changes invalidate them.
// Pattern: Presentation transition regression tests.
// Usage: go test ./internal/tui/viewport -run 'TestUnchanged|TestChangedPresentation'.
// Related: internal/tui/viewport/update.go, internal/tui/viewport/details.go, internal/tui/viewport/window.go.
package viewport

import (
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestUnchangedBlockUpdateRetainsVisibleWindow(t *testing.T) {
	value := Block{Text: "**same**", Markdown: true, Streaming: true, SpaceAfter: 2}
	view := View{Follow: true}
	view.AppendBlock(value)
	var layout Transcript
	layout.Prepare(80, &view, nil)
	before := layout.Window(0, layout.Count())
	if !view.Update(view.Next, value) || view.dirtyID != 0 {
		t.Fatal("unchanged content invalidated the display")
	}
	layout.Prepare(80, &view, nil)
	after := layout.Window(0, layout.Count())
	if len(after) != len(before) || &after[0] != &before[0] || !view.Blocks[0].Streaming {
		t.Fatal("unchanged update replaced the cached window or ended the stream")
	}
}

func TestUnchangedCollapsedDetailKeepsPreparedLayout(t *testing.T) {
	value := Block{Text: "full result", Summary: "brief", Details: "full result", DetailRef: &runrecord.EventReference{Session: "original", Sequence: 1}}
	view := View{Follow: true, Collapsed: true}
	view.AppendBlock(value)
	var layout Transcript
	layout.Prepare(80, &view, nil)
	layout.Window(0, layout.Count())
	if !view.Update(view.Next, value) || view.dirtyID != 0 || !view.Blocks[0].layout.ready {
		t.Fatal("unchanged collapsed result discarded its prepared layout")
	}
	if view.Blocks[0].Text != "brief" || view.Blocks[0].Details != "" {
		t.Fatal("no-op detection restored hidden details")
	}
}

func TestChangedPresentationStillInvalidates(t *testing.T) {
	for name, change := range map[string]func(*Block){
		"text":      func(b *Block) { b.Text = "different" },
		"summary":   func(b *Block) { b.Summary = "summary" },
		"details":   func(b *Block) { b.Details = "details" },
		"reference": func(b *Block) { b.DetailRef = &runrecord.EventReference{Session: "new", Sequence: 1} },
		"changes":   func(b *Block) { b.Changes = &Changes{Source: b.Text} },
		"markdown":  func(b *Block) { b.Markdown = false },
		"thinking":  func(b *Block) { b.Thinking = true },
		"centered":  func(b *Block) { b.Centered = true },
		"streaming": func(b *Block) { b.Streaming = false },
		"pairing":   func(b *Block) { b.KeepWithNext = true },
		"spacing":   func(b *Block) { b.SpaceAfter = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			value := Block{Text: "**same**", Markdown: true, Streaming: true, SpaceAfter: 2}
			view := View{Follow: true}
			view.AppendBlock(value)
			var layout Transcript
			layout.Prepare(80, &view, nil)
			layout.Window(0, layout.Count())
			change(&value)
			if !view.Update(view.Next, value) || view.dirtyID != view.Next {
				t.Fatal("changed presentation was discarded")
			}
			if name == "streaming" && view.Blocks[0].stream.RetainedBytes() != 0 {
				t.Fatal("completion retained mutable stream state")
			}
		})
	}
}
