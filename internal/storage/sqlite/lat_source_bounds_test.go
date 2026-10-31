// Purpose: Bound independent-source hydration before large canonical records are materialized.
// Pattern: SQLite projection and integrity regression tests.
// Usage: go test ./internal/storage/sqlite -run TestLATSource.
// Related: internal/storage/sqlite/lat_sources.go, internal/storage/sqlite/lat_observation.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestLATSourceStopsBeforeUnneededLargeUserPayload(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	for _, text := range []string{"keep this", strings.Repeat("large requirement", 10000)} {
		if err = s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.db.Exec("UPDATE payloads SET body=X'00'"); err != nil {
		t.Fatal(err)
	}
	records, next, err := s.LATRecords(ctx, id, 0, 2, 100)
	if err != nil || len(records) != 1 || next != 1 || records[0].Text != "keep this" {
		t.Fatal("large later source was hydrated", next, err)
	}
	if _, _, err = s.LATRecords(ctx, id, 1, 2, 100); !errors.Is(err, latmemory.ErrSourceBudget) {
		t.Fatal("oversized mandatory source was silently dropped", err)
	}
}

func TestLATSourceAuthenticatesLargeOutputAndRestoresDiffText(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	e := runrecord.Event{Kind: "tool_succeeded", CallID: "write", Text: "actual output", Call: &toolapi.Call{ID: "write", Name: "write_file", Arguments: []byte(`{"path":"a","content":"x"}`)}, Output: json.RawMessage(`{"result":"` + strings.Repeat("x", 2<<20) + `"}`), ChangePreview: strings.Repeat("+한국어 변경\n", 10000), Operation: &toolapi.Operation{Name: "write_file", Status: "tool_succeeded", Result: "actual output"}}
	if err = s.RecordEvent(ctx, id, e); err != nil {
		t.Fatal(err)
	}
	records, _, err := s.LATRecords(ctx, id, 0, 1, 128<<10)
	if err != nil || len(records) != 1 || len(records[0].Text) > 32<<10 || !strings.Contains(records[0].Text, "한국어 변경") || !strings.Contains(records[0].Text, "original record retained") {
		t.Fatal("bad bounded observation", err)
	}
	if _, err = s.db.Exec("UPDATE payloads SET body=X'00'"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.LATRecords(ctx, id, 0, 1, 128<<10); err == nil {
		t.Fatal("corrupt omitted output was accepted")
	}
}

func TestLATSourceKeepsLargeCurrentRequestOutsideSmallEvidencePacket(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	current := strings.Repeat("current instruction ", 1000)
	for _, text := range []string{"earlier requirement", current} {
		if err = s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	c := latmemory.Controller{Repository: s, Session: id, Redact: func(text string) string { return text }}
	body, err := c.Tail(ctx, 2048, current)
	if err != nil || !strings.Contains(string(body), "earlier requirement") || strings.Contains(string(body), "current instruction") {
		t.Fatal("accepted current user instruction could not be excluded exactly", err)
	}
}
