// Purpose: Bound editable input recall by retained bytes as well as entry count.
// Pattern: Event-loop input-history regression tests.
// Usage: go test ./internal/tui/session -run TestInputHistoryByteBudget.
// Related: internal/tui/session/input_history.go, internal/storage/history/inputs.go.
package session

import (
	"fmt"
	"strings"
	"testing"
)

func TestInputHistoryByteBudgetKeepsNewestWholeInputs(t *testing.T) {
	var state requestState
	body := strings.Repeat("x", 256<<10)
	for index := range 200 {
		state.rememberInput(fmt.Sprintf("%03d%s", index, body))
	}
	bytes := 0
	for _, text := range state.inputs {
		bytes += len(text)
	}
	if bytes > 16<<20 || !strings.HasPrefix(state.inputs[len(state.inputs)-1], "199") {
		t.Fatalf("recall retained %d bytes or lost the newest input", bytes)
	}
}
