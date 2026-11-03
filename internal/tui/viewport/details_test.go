// Purpose: Verify the current detail preference applies to new and restored tool blocks.
// Pattern: Presentation behavior tests.
// Usage: go test ./internal/tui/viewport -run TestDetails.
// Related: internal/tui/viewport/details.go, internal/tui/viewport/viewport.go.
package viewport

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"testing"
	"unsafe"
)

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

func TestDetailsRejectChangedSourceAndApplyMatchingOriginal(t *testing.T) {
	reference := runrecord.EventReference{Session: "source", Sequence: 5}
	view := View{Follow: true}
	view.AppendBlock(Block{Text: "summary", Summary: "summary", DetailRef: &reference})
	id := view.Next
	if view.UpdateDetails(id, runrecord.EventReference{Session: "other", Sequence: 5}, Block{Text: "wrong"}) {
		t.Fatal("unrelated source replaced detail")
	}
	if !view.UpdateDetails(id, reference, Block{Text: "original detail"}) || view.Blocks[0].Text != "original detail" {
		t.Fatal("matching original was not applied")
	}
}

func TestDetailsBudgetIncludesDiffRangesAndKeepsReloadableSummary(t *testing.T) {
	reference := &runrecord.EventReference{Session: "source", Sequence: 1}
	changes := &Changes{Source: "detail", Ranges: make([]ChangeRange, sourceMemoryBudget/int(unsafe.Sizeof(ChangeRange{}))+1)}
	view := View{Follow: true}
	view.AppendBlock(Block{Summary: "summary", Details: "detail", DetailRef: reference, Changes: changes})
	if view.sourceBytes <= sourceMemoryBudget {
		t.Fatal("diff metadata was omitted from the memory budget")
	}
	id := view.Next
	view.Append("latest response", false)
	if len(view.Blocks) != 2 || view.Blocks[0].Details != "" || view.Blocks[0].Changes != nil || view.Blocks[0].DetailRef != reference {
		t.Fatal("reloadable detail was not released before dropping its history")
	}
	if view.sourceBytes > sourceMemoryBudget || !view.UpdateDetails(id, *reference, Block{Text: "reloaded"}) {
		t.Fatal("released detail cannot be reloaded")
	}
}
