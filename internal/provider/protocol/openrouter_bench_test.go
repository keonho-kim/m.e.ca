// Purpose: Measure allocation cost while assembling fragmented OpenRouter reasoning.
// Pattern: Protocol microbenchmark.
// Usage: go test ./internal/provider/protocol -run '^$' -bench BenchmarkOpenRouterReasoning -benchmem.
// Related: internal/provider/protocol/chat/reasoning.go.
package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/protocol/chat"
)

func BenchmarkOpenRouterReasoningFragments(b *testing.B) {
	fragment := json.RawMessage(`{"type":"reasoning.text","text":"` + strings.Repeat("x", 64) + `","index":0}`)
	b.SetBytes(1000 * 64)
	for b.Loop() {
		state := chat.ReasoningState{}
		for i := 0; i < 1000; i++ {
			if _, err := state.Add([]json.RawMessage{fragment}, true); err != nil {
				b.Fatal(err)
			}
		}
		if _, err := state.Continuation(); err != nil {
			b.Fatal(err)
		}
	}
}
