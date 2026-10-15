// Purpose: Keep main-agent context focused on work facts rather than review or execution controls.
// Pattern: Context projection regression tests.
// Usage: go test ./internal/agent/run -run TestTaskContext.
// Related: internal/agent/run/task_context.go, internal/agent/prompt/defaults/system.md.
package run

import (
	"github.com/keonho-kim/v-code/internal/agent/completion"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"strings"
	"testing"
)

func TestTaskContextContainsWorkWithoutReviewInternals(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	s := execution{runner: r, limits: DefaultLimits(), system: prompt.Defaults().System, memoryContext: "Earlier work on a", checkpoint: Checkpoint{Objective: "Finish a", Work: completion.State{Reviewed: "PRIVATE_REVIEW_HASH", Verdict: &completion.Verdict{Decision: "continue", Reason: "PRIVATE_REVIEW_REASON", Unmet: []string{"Create b"}}, Receipts: []completion.Receipt{{ID: "PRIVATE_CALL_ID", Tool: "write_file", Targets: []string{"a"}, Success: true, Output: "Created a"}}}}}
	request, err := s.request("Continue the requested work", inference.Agent)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, m := range request.Messages {
		text.WriteString(m.Content)
	}
	for _, private := range []string{"PRIVATE_REVIEW_HASH", "PRIVATE_REVIEW_REASON", "PRIVATE_CALL_ID", "judgement_current", "pending_review", "checkpoint", "Conclave"} {
		if strings.Contains(text.String(), private) {
			t.Fatalf("exposed control information %s", private)
		}
	}
	for _, work := range []string{"Finish a", "Created a", "Create b"} {
		if !strings.Contains(text.String(), work) {
			t.Fatalf("lost task information %s", work)
		}
	}
}
