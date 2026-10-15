// Purpose: Verify internal role Thinking policy respects confirmed OFF support and caller selections.
// Pattern: Pure parameter contract tests.
// Usage: go test ./internal/agent/inference.
// Related: internal/agent/inference/settings.go.
package inference

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"testing"
)

func TestRoleThinkingHonorsConfirmedSupport(t *testing.T) {
	on := true
	original := modelapi.Request{Thinking: &on, ReasoningEffort: "high"}
	p := Defaults().Conclave
	for _, supported := range []bool{false, true} {
		got := p.ApplyThinking(original, supported)
		if supported && (*got.Thinking || got.ReasoningEffort != "") {
			t.Fatal("review did not disable supported Thinking")
		}
		if !supported && (!*got.Thinking || got.ReasoningEffort != "high") {
			t.Fatal("unsupported OFF changed selection")
		}
	}
	got := Defaults().Agent.ApplyThinking(original, true)
	if !*got.Thinking || got.ReasoningEffort != "high" || !on {
		t.Fatal("main selection mutated")
	}
}
