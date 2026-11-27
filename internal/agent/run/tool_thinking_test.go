// Purpose: Preserve finalized visible reasoning when a streamed reply becomes a native tool call.
// Pattern: Scripted generation-to-tool regression test.
// Usage: go test ./internal/agent/run -run TestToolTransitionKeepsThinking.
// Related: internal/agent/run/generate.go, internal/agent/run/codemode.go.
package run

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/modeltest"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestToolTransitionKeepsThinking(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	calls := 0
	r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		if calls > 1 {
			return emit(modelapi.Event{Delta: "Final answer", Done: true})
		}
		if err := emit(modelapi.Event{Thinking: "Stable reasoning", Streaming: true}); err != nil {
			return err
		}
		return emit(modelapi.Event{Done: true, Calls: modeltest.Calls([]toolapi.Call{{ID: "read", Name: "inspect_fixture", Arguments: []byte(`{"path":"a"}`)}})})
	})
	visible := ""
	generation := 0
	seen, toolStarted := false, false
	result, err := r.Run(t.Context(), "Inspect and report", approve, func(event record.Event) error {
		if event.Kind == "generation_started" {
			generation++
		}
		if generation == 1 && event.Kind == "stream" {
			visible = event.Thinking
			if seen && visible == "" {
				t.Error("finalized reasoning cleared before tool execution")
			}
			seen = seen || visible != ""
		}
		if event.Kind == "tool_started" {
			toolStarted = true
			if visible != "Stable reasoning" {
				t.Errorf("tool started with missing reasoning: %q", visible)
			}
		}
		return nil
	})
	if err != nil || result.Text != "Final answer" || !toolStarted {
		t.Fatalf("result=%+v toolStarted=%t error=%v", result, toolStarted, err)
	}
}
