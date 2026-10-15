// Purpose: Verify concise failure evidence is available to summarization without being appended to memory.
// Pattern: Compaction source and context contract test.
// Usage: go test ./internal/agent/compact -run TestFailedDiagnostic.
// Related: internal/agent/compact/controller.go, internal/agent/compact/context.go.
package compact

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"strings"
	"testing"
)

func TestFailedDiagnosticIsSourceOnly(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 200000, Usage: 1, Timeline: 1}, records: []Record{{Sequence: 1, Role: "outcome", Text: "execute_command failed: src/a:4 expected ready, got missing"}}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	if _, err := c.Refresh(context.Background(), func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if !strings.Contains(request.Messages[1].Content, "src/a:4 expected ready, got missing") {
			t.Fatal("execution evidence missing from summary source")
		}
		return emit(modelapi.Event{Delta: summary(), Done: true})
	}); err != nil {
		t.Fatal(err)
	}
	text, _, err := c.Context()
	if err != nil || text != summary() {
		t.Fatalf("unexpected diagnostic appendix: %s %v", text, err)
	}
}
