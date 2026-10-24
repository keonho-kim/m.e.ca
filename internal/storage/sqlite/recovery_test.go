// Purpose: Verify session listing, model settings and interrupted Tool Calls recover without replay.
// Pattern: Repository behavior tests.
// Usage: go test ./internal/storage/sqlite -run TestResume.
// Related: internal/storage/recovery/recovery.go, internal/storage/sqlite/selection.go.
package sqlite

import (
	"context"
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/recovery"
	"path/filepath"
	"testing"
)

func TestResumeSettingsAndInterruptedTool(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	session := s.Session()
	for _, m := range []modelapi.Message{{Role: "user", Content: "fix"}, {Role: "assistant", Calls: []toolapi.Call{{ID: "pending", Name: "write_file", Arguments: json.RawMessage(`{"path":"file","content":"new"}`)}}}} {
		if err := s.AppendMessage(ctx, session.ID, m); err != nil {
			t.Fatal(err)
		}
	}
	connection, err := modelconn.New(modelconn.Ollama, "")
	if err != nil {
		t.Fatal(err)
	}
	on := true
	if err = s.SaveSelection(ctx, session.ID, modelconn.Selection{Connection: connection, Model: "qwen", Thinking: &on, Details: &modelconn.ModelInfo{ID: "qwen", Reasoning: &modelconn.Reasoning{On: true, Off: true}}}); err != nil {
		t.Fatal(err)
	}
	if err = recovery.Restore(ctx, s, session.ID); err != nil {
		t.Fatal(err)
	}
	if err = recovery.Restore(ctx, s, session.ID); err != nil {
		t.Fatal(err)
	}
	history, err := s.Messages(ctx, session.ID, 100)
	if err != nil || len(history) != 3 || history[2].ToolCallID != "pending" {
		t.Fatalf("recovery: %+v %v", history, err)
	}
	catalog, err := OpenSessions(filepath.Dir(filepath.Dir(s.path)))
	if err != nil {
		t.Fatal(err)
	}
	list, err := catalog.Conversations(ctx, "")
	if err != nil || len(list) != 1 || list[0].Summary != "fix" {
		t.Fatalf("list: %+v %v", list, err)
	}
	selected, err := s.Selection(ctx, session.ID)
	if err != nil || selected.Thinking == nil || !*selected.Thinking {
		t.Fatalf("Thinking lost: %+v %v", selected, err)
	}
}
