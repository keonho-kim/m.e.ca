// Purpose: Verify attributed headers and bodies survive bounded history as one display group.
// Pattern: Presentation retention regression tests.
// Usage: go test ./internal/tui/viewport -run TestAttributed.
// Related: internal/tui/viewport/viewport.go, internal/tui/viewport/update.go.
package viewport

import (
	"strconv"
	"strings"
	"testing"
)

func TestAttributedHistoryKeepsPairsWhileFollowingOrReading(t *testing.T) {
	for _, follow := range []bool{true, false} {
		view := View{Follow: follow}
		for i := range 510 {
			text := strconv.Itoa(i)
			view.AppendBlock(Block{Text: "sender " + text, KeepWithNext: true})
			view.AppendBlock(Block{Text: "body " + text})
		}
		assertAttributedPairs(t, view)
		if view.Blocks[len(view.Blocks)-1].Text != "body 509" {
			t.Fatal("newest message was evicted while reading")
		}
	}
}

func TestAttributedHistoryPrependAndMemoryPressureKeepPairs(t *testing.T) {
	view := View{Follow: true}
	for range 500 {
		view.AppendBlock(Block{Text: "sender", KeepWithNext: true})
		view.AppendBlock(Block{Text: "body"})
	}
	view.Prepend([]Block{{Text: "old sender", KeepWithNext: true}, {Text: "old body"}})
	assertAttributedPairs(t, view)
	if view.Blocks[0].Text != "old sender" {
		t.Fatal("newly loaded oldest message was evicted")
	}
	payload := strings.Repeat("x", sourceMemoryBudget/2)
	view = View{Follow: true}
	for range 2 {
		view.AppendBlock(Block{Text: "sender", KeepWithNext: true})
		view.AppendBlock(Block{Text: payload})
	}
	assertAttributedPairs(t, view)
	if len(view.Blocks) != 2 {
		t.Fatal("memory pressure retained an orphaned sender or body")
	}
	view.Update(view.Blocks[1].ID, Block{Text: payload + payload})
	assertAttributedPairs(t, view)
}

func assertAttributedPairs(t *testing.T, view View) {
	t.Helper()
	if len(view.Blocks)%2 != 0 {
		t.Fatalf("history retained an incomplete message: %d blocks", len(view.Blocks))
	}
	for i := 0; i < len(view.Blocks); i += 2 {
		if !view.Blocks[i].KeepWithNext || view.Blocks[i+1].KeepWithNext {
			t.Fatalf("attribution separated at block %d", view.Blocks[i].ID)
		}
	}
}
