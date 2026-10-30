// Purpose: Restore editor input and pending calls independently of large or unavailable original payloads.
// Pattern: Session repository regression tests.
// Usage: SQLite tests verify selective hydration, ordering and cancellation during recovery.
// Related: internal/storage/sqlite/input_history.go, internal/storage/sqlite/pending_tools.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/recovery"
)

func TestRestorationSkipsCompletedPayloads(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx, id := context.Background(), s.Session().ID
	picture, err := media.New("shot.png", "image/png", "png", 16, 16, bytes.Repeat([]byte("snapshot"), 8192))
	if err != nil {
		t.Fatal(err)
	}
	call := func(id, content string) toolapi.Call {
		body, err := json.Marshal(map[string]string{"path": "source", "content": content})
		if err != nil {
			t.Fatal(err)
		}
		return toolapi.Call{ID: id, Name: "write_file", Arguments: body}
	}
	resolved := call("reused", strings.Repeat("completed", 8192))
	first, reused, last := call("first", "first"), call("reused", "new operation"), call("last", "last")
	messages := []modelapi.Message{
		{Role: "user", Content: "older", Images: []media.Image{picture}},
		{Role: "assistant", Calls: []toolapi.Call{resolved}},
		{Role: "tool", ToolCallID: resolved.ID, ToolName: resolved.Name, Images: []media.Image{picture}},
		{Role: "assistant", Calls: []toolapi.Call{first}},
	}
	for range 1100 {
		messages = append(messages, modelapi.Message{Role: "assistant", Content: "observation"})
	}
	messages = append(messages,
		modelapi.Message{Role: "user", Content: "latest"},
		modelapi.Message{Role: "assistant", Calls: []toolapi.Call{reused, resolved, last}},
		modelapi.Message{Role: "tool", ToolCallID: resolved.ID, ToolName: resolved.Name},
	)
	// Use a distinct completed ID in the last batch; the new "reused" call must
	// not be confused with the old result from the earlier generation.
	messages[len(messages)-2].Calls[1].ID = "finished"
	messages[len(messages)-1].ToolCallID = "finished"
	entries := make([]runrecord.Entry, len(messages))
	for i := range messages {
		entries[i].Message = &messages[i]
	}
	if err := s.Commit(ctx, id, entries); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE payloads SET body=x'00'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ContextMessages(ctx, id, 0, 16*1024*1024, true); err == nil {
		t.Fatal("test fixture did not make original payloads unavailable")
	}
	inputs, err := s.RecentInputs(ctx, id, 2)
	if err != nil || !slices.Equal(inputs, []string{"older", "latest"}) {
		t.Fatalf("input projection loaded binary state or lost older input: %v %v", inputs, err)
	}
	inputs, err = s.RecentInputs(ctx, id, 1)
	if err != nil || !slices.Equal(inputs, []string{"latest"}) {
		t.Fatalf("input limit: %v %v", inputs, err)
	}
	pending, err := s.PendingTools(ctx, id)
	if err != nil || len(pending) != 3 {
		t.Fatalf("pending calls: %d %v", len(pending), err)
	}
	for index, want := range []toolapi.Call{first, reused, last} {
		if pending[index].ID != want.ID || !bytes.Equal(pending[index].Arguments, want.Arguments) {
			t.Fatalf("pending order or arguments changed at %d", index)
		}
	}
	if err := recovery.Restore(ctx, s, id); err != nil {
		t.Fatal(err)
	}
	if pending, err := s.PendingTools(ctx, id); err != nil || len(pending) != 0 {
		t.Fatalf("recovery left pending calls: %v %v", pending, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.RecentInputs(cancelled, id, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("input cancellation lost: %v", err)
	}
	if _, err := s.PendingTools(cancelled, id); !errors.Is(err, context.Canceled) {
		t.Fatalf("recovery cancellation lost: %v", err)
	}
}

func TestPendingToolArgumentsAreRestored(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx, id := context.Background(), s.Session().ID
	arguments, _ := json.Marshal(map[string]string{"path": "source", "content": strings.Repeat("pending source\n", 8192)})
	call := toolapi.Call{ID: "pending", Name: "write_file", Arguments: arguments}
	if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{call}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.PendingTools(ctx, id)
	if err != nil || len(got) != 1 || !bytes.Equal(got[0].Arguments, arguments) {
		t.Fatal("pending arguments were not restored exactly", err)
	}
}
