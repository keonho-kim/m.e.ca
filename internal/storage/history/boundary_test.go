// Purpose: Verify recent-tail budgets retain complete native call and result groups.
// Pattern: Pure history-selection tests.
// Usage: go test ./internal/storage/history -run TestTail.
// Related: internal/storage/history/boundary.go.
package history

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"testing"
)

func TestTailRetainsAllCallsAndResultsPastByteTarget(t *testing.T) {
	s := TailSelection{Target: 10}
	for i, id := range []string{"b", "a"} {
		if _, ready := s.Keep(int64(5-i), 20, modelapi.Message{Role: "tool", ToolCallID: id}); ready {
			t.Fatal("split call from result")
		}
	}
	boundary, ready := s.Keep(3, 20, modelapi.Message{Role: "assistant", Calls: []tool.Call{{ID: "a"}, {ID: "b"}}})
	if !ready || boundary != 2 {
		t.Fatalf("boundary=%d ready=%v", boundary, ready)
	}
}

func TestTailExcludesOlderOversizedGroupWhenRecentPairFits(t *testing.T) {
	s := TailSelection{Target: 1024}
	messages := []modelapi.Message{
		{Role: "tool", ToolCallID: "recent"},
		{Role: "assistant", Calls: []tool.Call{{ID: "recent"}}},
		{Role: "tool", ToolCallID: "large"},
		{Role: "assistant", Calls: []tool.Call{{ID: "large"}}},
	}
	for i, m := range messages {
		size := 100
		if i == 2 {
			size = 32000
		}
		boundary, ready := s.Keep(int64(4-i), size, m)
		if i < 3 && ready {
			t.Fatal("split a native group")
		}
		if i == 3 && (!ready || boundary != 2) {
			t.Fatalf("older large output retained: boundary=%d ready=%v", boundary, ready)
		}
	}
}
