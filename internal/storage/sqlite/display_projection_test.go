// Purpose: Verify terminal history never hydrates hidden call bodies or blocks failed-call records.
// Pattern: Persistence projection and failure isolation tests.
// Usage: go test ./internal/storage/sqlite -run TestDisplayProjection.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestDisplayProjectionDoesNotReadHiddenArguments(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	call := toolapi.Call{ID: "call_test", Name: "write_file", Arguments: json.RawMessage(`{"path":"file.txt","content":"` + strings.Repeat("x", 100000) + `"}`)}
	if err = s.RecordEvent(ctx, id, runrecord.Event{Kind: "tool_succeeded", CallID: call.ID, Call: &call, Text: "Written"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE payloads SET body=X'00'"); err != nil {
		t.Fatal(err)
	}
	rows, _, err := s.Timeline(ctx, id, 0, 100)
	if err != nil || len(rows) != 1 || rows[0].Event.Call == nil {
		t.Fatalf("hidden payload was hydrated: %v", err)
	}
	args, err := toolapi.DisplayArgs(rows[0].Event.Call.Arguments)
	if err != nil || args.Path != "file.txt" || len(rows[0].Event.Call.Arguments) > 128 {
		t.Fatalf("display argument projection lost: %+v %v", args, err)
	}
	var out bytes.Buffer
	if err = s.Export(ctx, id, &out); err == nil {
		t.Fatal("original export ignored hidden payload corruption")
	}
}

func TestDisplayProjectionRecordsRejectedArgumentTypes(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	call := toolapi.Call{ID: "call_test", Name: "write_file", Arguments: json.RawMessage(`{"path":123}`)}
	if err = s.RecordEvent(context.Background(), s.Session().ID, runrecord.Event{Kind: "tool_failed", CallID: call.ID, Call: &call, Text: "path must be text"}); err != nil {
		t.Fatal("invalid display fields prevented recording the tool failure", err)
	}
}
