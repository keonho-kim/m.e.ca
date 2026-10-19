// Purpose: Verify long tool sequences end with the model's own final report.
// Pattern: Behavioral regression tests with scripted model replies.
// Usage: go test ./internal/agent/run -run TestToolSequence.
// Related: internal/agent/run/run.go, internal/agent/run/batch.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestToolSequenceAllowsModelReportAfter128Calls(t *testing.T) {
	for _, final := range []string{
		"The requested work is complete.",
		"I cannot complete the remaining change: the required service is unavailable. The completed files are preserved; provide access to continue.",
	} {
		g, e, j := &generator{}, &executor{}, &journal{}
		for index := range 130 {
			g.replies = append(g.replies, nativeCall("inspect_fixture", map[string]any{"path": fmt.Sprintf("file-%d", index)}))
		}
		g.replies = append(g.replies, final)
		result, err := fixture(g, e, j).Run(context.Background(), "Inspect the required files", approve, emit)
		if err != nil || result.Status != Answered || result.Text != final || e.executed != 130 || len(g.requests) != 131 {
			t.Fatalf("result=%+v tools=%d model_calls=%d err=%v", result, e.executed, len(g.requests), err)
		}
		if got := j.messages[len(j.messages)-1].Content; got != final {
			t.Fatalf("final report changed: %q", got)
		}
	}
}

type pairedSequence struct{ rounds int }

func (g *pairedSequence) Generate(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	g.rounds++
	if g.rounds > 65 {
		return emit(modelapi.Event{Delta: "All required observations are available.", Done: true})
	}
	var calls []toolapi.Call
	for index := range 2 {
		calls = append(calls, toolapi.Call{
			ID:        fmt.Sprintf("call-%d-%d", g.rounds, index),
			Name:      "inspect_fixture",
			Arguments: json.RawMessage(fmt.Sprintf(`{"path":"file-%d-%d"}`, g.rounds, index)),
		})
	}
	return emit(modelapi.Event{Calls: calls, Done: true})
}

func TestToolSequenceBatchCanCross128Calls(t *testing.T) {
	g, e, j := &pairedSequence{}, &executor{}, &journal{}
	r := fixture(&generator{}, e, j)
	r.Generator = g
	result, err := r.Run(context.Background(), "Inspect the required pairs", approve, emit)
	if err != nil || result.Status != Answered || e.executed != 130 || g.rounds != 66 {
		t.Fatalf("result=%+v tools=%d model_calls=%d err=%v", result, e.executed, g.rounds, err)
	}
}
