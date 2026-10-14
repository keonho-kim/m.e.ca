// Purpose: Verify compaction uses its own configurable generation settings for each summary call.
// Pattern: Compaction behavior test.
// Usage: go test ./internal/agent/compact -run TestInferenceCompaction.
// Related: internal/agent/compact/summarize.go, internal/agent/inference/settings.go.
package compact

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"testing"
)

func TestInferenceCompaction(t *testing.T) {
	roles := inference.Defaults()
	roles.Compaction.Temperature = 0.15
	roles.Compaction.MaxOutputTokens = 5000
	repo := &repository{mark: Watermark{Tokens: 256000, Usage: 1, Timeline: 1}, records: []Record{{Sequence: 1, Role: "user", Text: "preserve this objective"}}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Inference: &roles, Redact: func(s string) string { return s }}
	calls := 0
	changed, err := c.Refresh(context.Background(), func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		if r.Temperature == nil || *r.Temperature != 0.15 || r.MaxOutput != 5000 {
			t.Fatalf("wrong summary settings: %+v", r)
		}
		return emit(modelapi.Event{Delta: summary(), Done: true})
	})
	if err != nil || !changed || calls == 0 {
		t.Fatalf("compaction: %t/%d %v", changed, calls, err)
	}
}
