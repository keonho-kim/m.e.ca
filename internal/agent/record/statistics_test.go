// Purpose: Reject unmeasured or invalid final execution statistics.
// Pattern: Execution record contract tests.
// Usage: go test ./internal/agent/record.
// Related: internal/agent/record/statistics.go, internal/agent/record/event_validation.go.
package record

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"testing"
)

func TestStatisticsCannotClaimUnmeasuredThroughput(t *testing.T) {
	for _, stats := range []Statistics{{ElapsedMillis: -1}, {ElapsedMillis: 1, GenerationMillis: 2}, {ElapsedMillis: 1, GenerationMillis: 1}, {Usage: &modelapi.TokenUsage{Input: -1}}} {
		if err := stats.check(); err == nil {
			t.Fatalf("invalid measurements accepted: %+v", stats)
		}
	}
	event := Event{Kind: "thinking", Statistics: &Statistics{}}
	if err := event.Check(); err == nil {
		t.Fatal("nonterminal event accepted request statistics")
	}
}
