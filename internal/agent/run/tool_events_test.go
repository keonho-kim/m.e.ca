// Purpose: Verify durable tool events carry the executed parameters and actual output.
// Pattern: Task contract tests.
// Usage: go test ./internal/agent/run -run TestToolEvents.
// Related: internal/agent/run/execute.go, internal/agent/run/events.go.
package run

import (
	"context"
	"encoding/json"
	"testing"
)

func TestToolEventsCarryCallAndOutput(t *testing.T) {
	g, e, j := &generator{replies: []string{"TOOL read_file", "README.md", "WAIT\nreview"}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	if _, err := r.Run(context.Background(), "inspect", false, approve, emit); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range j.events {
		if event.Kind != "tool_succeeded" {
			continue
		}
		body, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		var fields struct {
			Call struct {
				Name      string
				Arguments map[string]string
			}
			Output json.RawMessage
		}
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Fatal(err)
		}
		if fields.Call.Name != "read_file" || fields.Call.Arguments["path"] != "README.md" || string(fields.Output) != `{"ok":true}` {
			t.Fatalf("tool details missing: %s", body)
		}
		found = true
	}
	if !found {
		t.Fatal("completion event missing")
	}
}
