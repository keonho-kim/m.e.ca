// Purpose: Verify each provider projects its declared reasoning field into model events.
// Pattern: Provider stream contract tests.
// Usage: go test ./internal/provider/protocol -run TestProviderReasoningFields.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/wire.go.
package protocol

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"testing"
)

func TestProviderReasoningFields(t *testing.T) {
	for _, test := range []struct {
		provider modelconn.ProviderKind
		body     string
	}{
		{modelconn.VLLM, `{"choices":[{"message":{"reasoning":"observed reasoning","content":"hello"},"finish_reason":"stop"}]}`},
		{modelconn.SGLang, `{"choices":[{"message":{"reasoning_content":"observed reasoning","content":"hello"},"finish_reason":"stop"}]}`},
		{modelconn.LiteLLM, `{"choices":[{"message":{"reasoning_content":"observed reasoning","content":"hello"},"finish_reason":"stop"}]}`},
		{modelconn.Ollama, `{"message":{"thinking":"observed reasoning","content":"hello"},"done":true}`},
	} {
		t.Run(string(test.provider), func(t *testing.T) {
			state := streamState{provider: test.provider}
			var answer, thinking string
			err := state.consume([]byte(test.body), true, func(event modelapi.Event) error { answer += event.Delta; thinking += event.Thinking; return nil })
			if err != nil || answer != "hello" || thinking != "observed reasoning" {
				t.Fatalf("answer=%q thinking=%q error=%v", answer, thinking, err)
			}
		})
	}
}
