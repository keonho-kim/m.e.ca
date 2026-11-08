// Purpose: Reject protected credentials hidden by JSON escapes in native provider state.
// Pattern: Scripted execution and journal regression test.
// Usage: go test ./internal/agent/run -run TestProtectedContinuation.
// Related: internal/agent/run/generate.go, internal/config/credentials/json.go.
package run

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/config/credentials"
)

func TestProtectedContinuationCannotPersistEscapedCredential(t *testing.T) {
	for _, state := range []string{
		`[{"type":"reasoning","summary":[{"text":"\u0070rovider-secret"}]}]`,
		`[{"type":"function_call","arguments":"{\"path\":\"\\u0070rovider-secret\"}"}]`,
	} {
		vault, journal := &credentials.Vault{}, &journal{}
		vault.Remember("provider-secret")
		r := fixture(&generator{}, &executor{}, journal)
		r.Redact, r.NewStreamMask = vault.Redact, vault.NewStreamMask
		r.Generator = boundaryModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			return emit(modelapi.Event{Delta: "answer", ReasoningState: []byte(state), Done: true})
		})
		if _, err := r.Run(t.Context(), "inspect", approve, emit); err == nil {
			t.Fatal("escaped credential was accepted")
		}
		for _, m := range journal.messages {
			if len(m.ReasoningState) > 0 {
				t.Fatal("protected native state persisted")
			}
		}
	}
}
