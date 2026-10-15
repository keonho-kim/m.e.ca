// Purpose: Verify the active user request remains the final task anchor after history and compaction.
// Pattern: Context contract tests.
// Usage: go test ./internal/agent/context -run TestActiveRequest.
// Related: internal/agent/context/request.go.
package context

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

func TestActiveRequestAlwaysEndsDecisionContext(t *testing.T) {
	objective := "Create only the requested directories and empty files."
	for _, history := range [][]modelapi.Message{
		nil,
		{{Role: "user", Content: objective}, {Role: "assistant", Content: "created empty files"}},
	} {
		request, err := Build(Input{System: "rules", Objective: objective, Memory: "Old task: implement an unrelated feature.", Messages: history, Question: "Use observed outcomes", Redact: func(s string) string { return s }, MaxBytes: 20000})
		if err != nil {
			t.Fatal(err)
		}
		last := request.Messages[len(request.Messages)-1].Content
		if !strings.HasSuffix(last, objective) {
			t.Fatalf("latest request is not the final anchor: %s", last)
		}
	}
}
