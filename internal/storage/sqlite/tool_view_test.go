// Purpose: Keep tool request and result hydration independent under large or damaged hidden payloads.
// Pattern: Persistence projection contract tests.
// Usage: go test -race ./internal/storage/sqlite -run TestToolView.
// Related: internal/storage/sqlite/tool_detail.go, internal/storage/sqlite/events.go.
package sqlite

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestToolViewHydratesOnlyItsSelectedSide(t *testing.T) {
	for _, requestLarge := range []bool{false, true} {
		name := "large-result"
		if requestLarge {
			name = "large-request"
		}
		t.Run(name, func(t *testing.T) {
			store, err := newTestStore(t)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			content, output := "request body", "result body"
			if requestLarge {
				content = strings.Repeat("request-body\n", 10000)
			} else {
				output = strings.Repeat("result-body\n", 10000)
			}
			args, err := json.Marshal(map[string]string{"path": "file.txt", "content": content})
			if err != nil {
				t.Fatal(err)
			}
			result, err := json.Marshal(map[string]string{"stdout": output})
			if err != nil {
				t.Fatal(err)
			}
			call := tool.Call{ID: "call", Name: "write_file", Arguments: args}
			if err := store.RecordEvent(t.Context(), store.Session().ID, record.Event{Kind: "tool_succeeded", Call: &call, CallID: call.ID, Output: result}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.db.Exec("UPDATE payloads SET body=X'00'"); err != nil {
				t.Fatal(err)
			}
			reference := record.EventReference{Session: store.Session().ID, Sequence: 1}
			if requestLarge {
				event, err := store.ToolEvent(t.Context(), reference, history.ReadBounds{})
				if err != nil || string(event.Output) != string(result) {
					t.Fatalf("result read hydrated request: %v", err)
				}
				if _, err := store.ToolRequest(t.Context(), reference, history.ReadBounds{}); err == nil {
					t.Fatal("damaged original request was accepted")
				}
			} else {
				event, err := store.ToolRequest(t.Context(), reference, history.ReadBounds{})
				if err != nil || event.Call == nil || string(event.Call.Arguments) != string(args) {
					t.Fatalf("request read hydrated result: %+v %v", event.Call, err)
				}
				if len(event.Output) != 0 {
					t.Fatal("request projection retained result")
				}
			}
		})
	}
}
