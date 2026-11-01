// Purpose: Verify LAT publishes native streaming and tool events without repeating the main draft.
// Pattern: Scripted execution and presentation-boundary regression.
// Usage: go test ./internal/agent/run -run TestLATDisplay.
// Related: internal/agent/run/lat_fork.go, internal/agent/run/lat.go.
package run

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestLATDisplayStreamsToolsAndSettlesMainBeforeReview(t *testing.T) {
	r, model, _, store := latFixture("Main answer")
	model.handlers["plan"] = func(ctx context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
		if len(model.requests["plan"]) == 1 {
			if err := send(modelapi.Event{Delta: "Inspecting sources", Thinking: "Review the file", Streaming: true}); err != nil {
				return err
			}
			var call toolapi.Call
			if err := json.Unmarshal([]byte(strings.TrimPrefix(nativeCall("inspect_fixture", map[string]any{"path": "api.py"}), "native:")), &call); err != nil {
				return err
			}
			return send(modelapi.Event{Calls: []toolapi.Call{call}, Done: true})
		}
		return send(modelapi.Event{Delta: "Plan ready", Streaming: true, Done: true})
	}
	var events []runrecord.Event
	result, err := r.Run(context.Background(), "request", approve, func(event runrecord.Event) error {
		events = append(events, event)
		return nil
	})
	if err != nil || result.Text != "Main answer" {
		t.Fatalf("run failed: %+v %v", result, err)
	}
	seen := map[string]bool{}
	draftAt, reviewAt, mainAnswers := -1, -1, 0
	for i, event := range events {
		if event.LAT != nil && event.LAT.Phase == "plan" {
			seen[event.Kind] = true
			if event.Kind == "stream" && event.Text != "" {
				seen["body_stream"] = true
			}
		}
		if event.LAT == nil && (event.Kind == "progress" || event.Kind == "response") && event.Text == "Main answer" {
			draftAt, mainAnswers = i, mainAnswers+1
		}
		if event.Kind == "lat_started" {
			reviewAt = i
		}
	}
	for _, kind := range []string{"lat_phase_started", "lat_phase_finished", "body_stream", "thinking", "progress", "tool_started", "tool_succeeded", "response"} {
		if !seen[kind] {
			t.Errorf("LAT hid %s", kind)
		}
	}
	if mainAnswers != 1 || draftAt < 0 || draftAt > reviewAt {
		t.Fatalf("main draft overwritten or repeated around review: answers=%d draft=%d review=%d", mainAnswers, draftAt, reviewAt)
	}
	copies := 0
	for _, message := range store.messages {
		if message.Role == "assistant" && strings.Contains(message.Content, "Main answer") {
			copies++
		}
	}
	if copies != 1 {
		t.Fatalf("main draft stored %d times", copies)
	}
}
