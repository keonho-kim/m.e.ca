// Purpose: Reproduce fenced JSON and missing evidence with one correction and recorded raw attempts.
// Pattern: Model-boundary contract tests.
// Usage: go test ./internal/agent/completion -run TestReviewCorrection.
// Related: internal/agent/completion/review.go, internal/agent/reviewjson/object.go.
package completion

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/reviewjson"
	"strings"
	"testing"
)

func TestReviewCorrectionPreservesFailureAndDoesNotInventEvidence(t *testing.T) {
	input := Input{Request: "create a", State: State{Receipts: []Receipt{{ID: "c", Tool: "write_file", Success: true}}}}
	for _, success := range []bool{true, false} {
		calls := 0
		var attempts []reviewjson.Attempt
		verdict, err := Review(context.Background(), input, modelapi.Request{}, func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
			calls++
			if len(r.Tools) != 0 {
				t.Fatal("review has tools")
			}
			answer := "```json\n{\"decision\":\"complete\",\"reason\":\"created\"}\n```"
			if calls == 2 {
				if !strings.Contains(r.Messages[len(r.Messages)-1].Content, "PreviousResponse") {
					t.Fatal("missing failed source in correction")
				}
				if success {
					answer = `{"decision":"complete","reason":"created","evidence":["c"]}`
				}
			}
			return emit(modelapi.Event{Delta: answer, Done: true})
		}, func(s string) string { return s }, func(a reviewjson.Attempt) error { attempts = append(attempts, a); return nil })
		if calls != 2 || len(attempts) != 2 || attempts[0].Error == "" || !strings.Contains(attempts[0].Raw, "```json") {
			t.Fatalf("missing diagnostics: %+v", attempts)
		}
		if success && (err != nil || verdict.Decision != "complete") || !success && err == nil {
			t.Fatalf("%+v %v", verdict, err)
		}
	}
}
