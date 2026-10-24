// Purpose: Verify compaction accepts large source instructions and summaries within the common body ceiling.
// Pattern: Compaction boundary regression tests.
// Usage: go test ./internal/agent/compact -run TestCapacity.
// Related: internal/agent/compact/usage.go, internal/agent/compact/summarize.go.
package compact

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"strings"
	"testing"
)

func TestCapacityLargeInstructionsAndSummary(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 245760, Usage: 1, Timeline: 1}, records: []Record{{Sequence: 1, Role: "user", Text: strings.Repeat("x", 70*1024)}}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	output := strings.Repeat("detail ", 4000)
	calls := 0
	changed, err := c.Refresh(context.Background(), nil, func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		return emit(modelapi.Event{Delta: output, Done: true})
	})
	if err != nil || !changed || calls != 1 || repo.memory.Markdown != output {
		t.Fatalf("changed=%v calls=%d error=%v", changed, calls, err)
	}
}
