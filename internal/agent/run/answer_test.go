// Purpose: Verify plain responses, JSON examples and empty replies terminate without tool execution.
// Pattern: Scripted conversation contract tests.
// Usage: go test ./internal/agent/run -run TestDirectAnswer.
// Related: internal/agent/run/run.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestDirectAnswer(t *testing.T) {
	for _, test := range []struct {
		name, request, answer string
	}{
		{"greeting", "Hello", "Hello"},
		{"JSON example", "Show an example JSON object", `{"tool":"example-only"}`},
		{"long answer", "Explain in detail", strings.Repeat("Detailed answer. ", 2048) + "End."},
		{"empty reply", "Hello", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := &generator{replies: []string{test.answer}}
			e := &executor{}
			result, err := fixture(g, e, &journal{}).Run(context.Background(), test.request, approve, emit)
			if len(g.requests) != 1 || e.prepared != 0 || e.executed != 0 || test.answer != "" && result.Text != test.answer {
				t.Fatalf("result=%+v requests=%d prepared=%d executed=%d err=%v", result, len(g.requests), e.prepared, e.executed, err)
			}
			if test.answer == "" {
				if err == nil || !strings.Contains(err.Error(), "empty") || result.Status != runrecord.Stopped {
					t.Fatalf("empty reply result=%+v err=%v", result, err)
				}
			} else if err != nil || result.Status != runrecord.Answered {
				t.Fatalf("answer result=%+v err=%v", result, err)
			}
		})
	}
}
