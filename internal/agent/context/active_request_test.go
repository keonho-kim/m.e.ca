// Purpose: Verify the active user request appears once alongside history and memory.
// Pattern: Context contract tests.
// Usage: go test ./internal/agent/context -run TestActiveRequest.
// Related: internal/agent/context/request.go.
package context

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"testing"
)

func TestActiveRequestAppearsOnceAfterCompaction(t *testing.T) {
	objective := "Create only the requested directories and empty files."
	for _, history := range [][]modelapi.Message{
		nil,
		{{Role: "user", Content: objective}, {Role: "assistant", Content: "created empty files"}},
	} {
		request, err := Build(Input{System: "rules", Objective: objective, Memory: "Old task: implement an unrelated feature.", Messages: history, Redact: func(s string) string { return s }, MaxBytes: 20000})
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, m := range request.Messages {
			if m.Role == "user" && m.Content == objective {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("active request count=%d: %+v", count, request.Messages)
		}

	}
}
