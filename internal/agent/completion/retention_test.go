// Purpose: Preserve earlier successful outcomes throughout a long request.
// Pattern: Completion state regression test.
// Usage: go test ./internal/agent/completion -run TestRetention.
// Related: internal/agent/completion/state.go.
package completion

import (
	"fmt"
	"testing"
)

func TestRetentionKeepsEarlierOutcomes(t *testing.T) {
	var state State
	for i := 0; i < 100; i++ {
		state.Observe(Receipt{ID: fmt.Sprint(i), Tool: "write_file", Targets: []string{fmt.Sprintf("/p/%d", i)}, Success: true}, nil, false)
	}
	if len(state.Receipts) != 100 || state.Receipts[0].ID != "0" {
		t.Fatal("earlier successful outcomes were discarded")
	}
	if err := state.Check(); err != nil {
		t.Fatal(err)
	}
}
