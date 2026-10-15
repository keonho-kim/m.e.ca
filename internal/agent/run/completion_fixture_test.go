// Purpose: Provide deterministic completion decisions to existing scripted model fixtures.
// Pattern: Test adapter.
// Usage: Scripted generators delegate only internal completion packets here.
// Related: internal/agent/run/run_test.go, internal/agent/run/completion_review_test.go.
package run

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
)

func completionFixture(r modelapi.Request, emit func(modelapi.Event) error) (bool, error) {
	text := r.Messages[len(r.Messages)-1].Content
	if !strings.HasPrefix(text, "REVIEW COMPLETION.") {
		return false, nil
	}
	var input completion.Input
	if err := json.Unmarshal([]byte(text[strings.Index(text, "\n")+1:]), &input); err != nil {
		return true, err
	}
	verdict := completion.Verdict{Decision: "blocked", Reason: "No successful operation was observed", Evidence: []string{"request"}}
	for i := len(input.State.Receipts) - 1; i >= 0; i-- {
		r := input.State.Receipts[i]
		if r.Success && !r.Stale {
			verdict = completion.Verdict{Decision: "complete", Reason: "Recorded outcomes satisfy the requested work", Evidence: []string{r.ID}}
			break
		}
	}
	if input.Trigger != "final answer proposal" && len(input.State.Receipts) > 0 {
		verdict = completion.Verdict{Decision: "continue", Reason: "Inspect the remaining original request using a different approach", Unmet: []string{input.Request}, Evidence: []string{"request"}}
	}
	body, err := json.Marshal(verdict)
	if err != nil {
		return true, err
	}
	return true, emit(modelapi.Event{Delta: string(body), Done: true})
}
