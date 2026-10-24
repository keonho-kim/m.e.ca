// Purpose: Verify measured tokenizer overflow refits one packet once and never publishes failed summaries.
// Pattern: Exact-count provider boundary simulation.
// Usage: go test ./internal/agent/compact -run TestSummaryMeasuredCapacity.
// Related: internal/agent/compact/summary_attempt.go, internal/agent/compact/budget.go.
package compact

import (
	"context"
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

func TestSummaryMeasuredCapacityRefitsOnce(t *testing.T) {
	for _, alwaysFail := range []bool{false, true} {
		prior := Memory{Revision: 1, Through: 1, Markdown: "Retain the agreed API."}
		repo := &repository{memory: prior, mark: Watermark{Timeline: 22, Usage: 10, Tokens: 250000}, records: []Record{{Sequence: 2, Role: "user", Text: "Preserve the request constraints"}}}
		for i := 3; i <= 22; i++ {
			repo.records = append(repo.records, Record{Sequence: int64(i), Role: "observation", Text: fmt.Sprintf("step %d: %s", i, strings.Repeat("x", 4000))})
		}
		c := Controller{ContextWindow: 65536, Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
		calls, firstBytes := 0, 0
		changed, err := c.Refresh(context.Background(), nil, func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
			calls++
			size := 0
			for _, m := range request.Messages {
				size += len(m.Content)
			}
			if calls == 1 {
				firstBytes = size
				return &modelapi.ContextLimitError{Input: size, Capacity: 32768}
			}
			if size >= firstBytes || size+request.MaxOutput > 32768 {
				t.Errorf("packet still exceeds measured budget: bytes=%d output=%d", size, request.MaxOutput)
			}
			for _, required := range []string{"Retain the agreed API", "Preserve the request constraints"} {
				if !strings.Contains(request.Messages[1].Content, required) {
					t.Errorf("refit lost %s", required)
				}
			}
			if alwaysFail {
				return &modelapi.ContextLimitError{Input: 40000, Capacity: 32768}
			}
			return emit(modelapi.Event{Delta: "- The requested work is ready to continue.", Done: true})
		})
		if calls != 2 || c.ContextWindow != 32768 {
			t.Fatalf("attempts=%d context=%d error=%v", calls, c.ContextWindow, err)
		}
		if alwaysFail {
			if err == nil || changed || repo.installed != 0 || repo.memory != prior {
				t.Fatal("failed correction replaced prior memory")
			}
		} else if err != nil || !changed || repo.installed != 1 {
			t.Fatalf("correction did not finish: changed=%v installs=%d error=%v", changed, repo.installed, err)
		}
	}
}
