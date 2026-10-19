// Purpose: Verify the current detail preference applies to new and restored tool blocks.
// Pattern: Presentation behavior tests.
// Usage: go test ./internal/tui/viewport -run TestDetails.
// Related: internal/tui/viewport/details.go, internal/tui/viewport/viewport.go.
package viewport

import "testing"

func TestDetailsPreferenceAcrossHistory(t *testing.T) {
	v := View{Follow: true}
	block := Block{Summary: "summary", Details: "full output"}
	v.AppendBlock(block)
	if v.Collapsed || v.Blocks[0].Text != block.Details {
		t.Fatal("default was not expanded")
	}
	v.ToggleDetails()
	v.AppendBlock(block)
	v.Prepend([]Block{block})
	for _, b := range v.Blocks {
		if b.Text != block.Summary {
			t.Fatal("new or historical output ignored collapse state")
		}
	}
	v.Append("agent answer", false)
	v.ToggleDetails()
	for _, b := range v.Blocks[:3] {
		if b.Text != block.Details {
			t.Fatal("original details lost")
		}
	}
	if v.Blocks[3].Text != "agent answer" {
		t.Fatal("agent answer was folded")
	}
}
