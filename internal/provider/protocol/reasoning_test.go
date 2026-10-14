// Purpose: Verify vLLM consumes its current reasoning contract without a legacy field fallback.
// Pattern: Provider stream contract tests.
// Usage: go test ./internal/provider/httpclient -run TestVLLMReasoningContentIsNotConsumed.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/wire.go.
package protocol

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"testing"
)

func TestVLLMReasoningContentIsNotConsumed(t *testing.T) {
	state := streamState{provider: modelconn.VLLM}
	var thinking string
	err := state.consume([]byte(`{"choices":[{"message":{"reasoning_content":"obsolete output","content":"hello"},"finish_reason":"stop"}]}`), true, func(event modelapi.Event) error {
		thinking += event.Thinking
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if thinking != "" {
		t.Fatalf("obsolete reasoning field consumed: %q", thinking)
	}
}
