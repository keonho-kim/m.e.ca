// Purpose: Check source identity when room and private payload numbers collide.
// Pattern: SQLite integration regression test.
// Usage: go test ./internal/storage/sqlite.
// Related: internal/storage/sqlite/team_references.go, internal/storage/sqlite/page_payloads.go.
package sqlite

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestTeamPayloadCachesKeepSourceIdentity(t *testing.T) {
	root, child := teamStorage(t)
	ctx := t.Context()
	shared := strings.Repeat("public ", 6000)
	appendPublic := func() {
		t.Helper()
		if _, err := root.AppendTeamMessage(ctx, teamchannel.Message{From: "Hana", Text: shared}); err != nil {
			t.Fatal(err)
		}
	}
	appendPublic()
	private := json.RawMessage(`{"private":"` + strings.Repeat("x", 40000) + `"}`)
	if err := child.RecordEvent(ctx, child.session.ID, runrecord.Event{Kind: "tool_succeeded", Text: "private result", Output: private}); err != nil {
		t.Fatal(err)
	}
	reference := &runrecord.EventReference{Session: child.session.ID, Sequence: child.latestEvent.Load()}
	if err := root.RecordEvent(ctx, root.session.ID, runrecord.Event{Kind: "team_reference", Agent: "Hana", Reference: reference}); err != nil {
		t.Fatal(err)
	}
	appendPublic()
	var output bytes.Buffer
	if err := root.Export(ctx, root.session.ID, &output); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	found := false
	for decoder.More() {
		var row struct {
			Kind string
			Body json.RawMessage
		}
		if err := decoder.Decode(&row); err != nil {
			t.Fatal(err)
		}
		if row.Kind != "event" {
			continue
		}
		var event runrecord.Event
		if err := json.Unmarshal(row.Body, &event); err != nil {
			t.Fatal(err)
		}
		if event.Kind != "tool_succeeded" {
			continue
		}
		found = true
		if !bytes.Equal(event.Output, private) {
			t.Fatalf("private output replaced by room payload: got %q, want prefix %q", string(event.Output[:min(70, len(event.Output))]), string(private[:70]))
		}
	}
	if !found {
		t.Fatal("private tool event not exported")
	}
}
