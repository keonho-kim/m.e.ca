// Purpose: Measure allocation cost while assembling fragmented OpenRouter reasoning.
// Pattern: Protocol microbenchmark.
// Usage: go test ./internal/provider/protocol -run '^$' -bench BenchmarkOpenRouterReasoning -benchmem.
// Related: internal/provider/protocol/openrouter.go.
package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func BenchmarkOpenRouterReasoningFragments(b *testing.B) {
	fragment := json.RawMessage(`{"type":"reasoning.text","text":"` + strings.Repeat("x", 64) + `","index":0}`)
	b.SetBytes(1000 * 64)
	for b.Loop() {
		state := streamState{}
		for i := 0; i < 1000; i++ {
			if _, err := state.routerDetails([]json.RawMessage{fragment}, true); err != nil {
				b.Fatal(err)
			}
		}
		if _, err := routerContinuation(state.reasoningDetails, true); err != nil {
			b.Fatal(err)
		}
	}
}
