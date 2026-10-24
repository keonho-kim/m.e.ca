// Purpose: Keep real conversation order without repeating the current user request.
// Pattern: Behavioral regression tests.
// Usage: go test ./internal/agent/context -run TestMinimal.
// Related: internal/agent/context/request.go.
package context

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"testing"
)

func TestMinimalContextKeepsOneRequestAndEndsWithToolResult(t *testing.T) {
	request, err := Build(Input{Objective: "Implement this", Messages: []modelapi.Message{
		{Role: "user", Content: "Implement this"},
		{Role: "assistant", Content: "Working"},
	}, Redact: func(s string) string { return s }, MaxBytes: 10000})
	if err != nil || len(request.Messages) != 3 || request.Messages[2].Content != "Working" {
		t.Fatalf("request=%+v err=%v", request, err)
	}
}
