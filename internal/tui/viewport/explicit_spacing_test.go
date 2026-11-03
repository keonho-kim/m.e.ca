// Purpose: Verify display-only spacing preserves source, internal blank rows and incremental layout ownership.
// Pattern: Presentation boundary regression tests.
// Usage: go test ./internal/tui/viewport -run TestExplicitSpacing.
// Related: internal/tui/viewport/block_layout.go, internal/tui/viewport/block_window.go, internal/tui/viewport/update.go.
package viewport

import (
	"reflect"
	"strings"
	"testing"
)

func TestExplicitSpacingDoesNotInventRowsForInvisibleMarkdown(t *testing.T) {
	for _, source := range []string{"[ref]: https://example.com", "[ref]: https://example.com\n"} {
		block := Block{Text: source, Markdown: true, SpaceAfter: 2}
		if rows := block.Window(80, 0, 5); len(rows) != 0 {
			t.Fatalf("invisible Markdown produced display padding: %+v", rows)
		}
		if block.Text != source {
			t.Fatal("original Markdown was changed")
		}
		block.Text += "\n[visible][ref]"
		rows := block.Window(80, 0, block.Count(80))
		if len(rows) != 3 || !strings.Contains(rows[0].Text, "visible") {
			t.Fatalf("later visible content was lost: %+v", rows)
		}
	}
}

func TestExplicitSpacingReplacesOnlyBoundaryBlanks(t *testing.T) {
	for _, markdown := range []bool{false, true} {
		source := "\n\nfirst\n\nlast\n\n\n\n"
		block := Block{Text: source, ID: 1, Markdown: markdown, SpaceAfter: 2}
		want := []string{"first", "", "last", "", ""}
		if count := block.Count(20); count != len(want) {
			t.Fatalf("markdown %v: got %d rows, want %d", markdown, count, len(want))
		}
		for start := range want {
			for count := 1; count <= len(want)-start; count++ {
				rows := block.Window(20, start, count)
				if len(rows) != count {
					t.Fatalf("markdown %v window %d/%d: %+v", markdown, start, count, rows)
				}
				for i, row := range rows {
					if row.Text != want[start+i] || row.ID != 1 {
						t.Fatalf("markdown %v window %d/%d row %d: %+v", markdown, start, count, i, row)
					}
				}
			}
		}
		if block.Text != source {
			t.Fatal("spacing rewrote original text")
		}
	}
}

func TestExplicitSpacingPreservesDiffColorsAndCopyOffsets(t *testing.T) {
	source := "\n  edit_file\n\n    +added\n\n\n"
	start := strings.Index(source, "    +added")
	block := Block{Text: source, SpaceAfter: 2, Changes: &Changes{
		Source: source, Ranges: []ChangeRange{{Start: start, End: start + len("    +added"), Kind: '+'}},
	}}
	rows := block.Window(80, 0, block.Count(80))
	if len(rows) != 5 || rows[2].Text != "    +added" || rows[2].Change != '+' || rows[2].Copy.Row.Start != start {
		t.Fatalf("spacing detached diff styles or copy offsets: %+v", rows)
	}
	if rows[3].Change != 0 || rows[4].Change != 0 {
		t.Fatal("display padding inherited a diff background")
	}
}

func TestExplicitSpacingInvalidatesLayoutAndTransfersStreamCache(t *testing.T) {
	output := View{Follow: true}
	output.AppendBlock(Block{Text: "**result**", Markdown: true, Streaming: true, SpaceAfter: 1})
	var layout Transcript
	layout.Prepare(20, &output, nil)
	layout.Window(0, layout.Count())
	output.Update(output.Next, Block{Text: "**result**", Markdown: true, Streaming: true, SpaceAfter: 2})
	layout.Prepare(20, &output, nil)
	before := layout.Window(0, layout.Count())
	if len(before) != 3 {
		t.Fatalf("spacing-only update reused stale rows: %+v", before)
	}
	output.Update(output.Next, Block{Text: "**result**", Markdown: true, SpaceAfter: 2})
	block := &output.Blocks[0]
	if len(block.layout.parsed) == 0 || block.stream.RetainedBytes() != 0 || block.layout.metrics != nil {
		t.Fatal("completion lost the reusable cache or retained mutable stream state")
	}
	layout.Prepare(20, &output, nil)
	if after := layout.Window(0, layout.Count()); !reflect.DeepEqual(before, after) {
		t.Fatalf("completion changed spacing: %+v", after)
	}
}

func TestExplicitSpacingOnlyWindowKeepsColdMarkdownReleased(t *testing.T) {
	for _, source := range []string{"**result**", strings.Repeat("**result**\n\n", 12000)} {
		block := Block{Text: source, Markdown: true, SpaceAfter: 2}
		count := block.Count(80)
		block.releaseParsed()
		rows := block.Window(80, count-2, 2)
		if len(rows) != 2 || rows[0].Text != "" || rows[1].Text != "" {
			t.Fatalf("cold boundary window lost spacing: %+v", rows)
		}
		if block.layout.parsed != nil || block.layout.packed != nil {
			t.Fatal("viewing only display padding reparsed the hidden document")
		}
		if rows := block.Window(80, 0, 1); len(rows) != 1 || rows[0].Text != "result" {
			t.Fatalf("cold source no longer restores: %+v", rows)
		}
	}
}
