// Purpose: Preserve authenticated compaction content and resolve cached private envelopes without reopening files.
// Pattern: Temporary SQLite and pure projection regression tests.
// Usage: go test ./internal/storage/sqlite -run 'TestTeamPage|TestSharedContext'.
// Related: internal/storage/sqlite/context_record.go, internal/storage/sqlite/team_references.go.
package sqlite

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestTeamPageCachedEnvelopeNeedsNoPrivateConnection(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	if err := child.RecordEvent(ctx, child.session.ID, runrecord.Event{Kind: "tool_succeeded", Text: "created", Output: []byte(`{"ok":true}`)}); err != nil {
		t.Fatal(err)
	}
	event := runrecord.Event{Kind: "team_reference", Agent: "Hana", Reference: &runrecord.EventReference{Session: child.session.ID, Sequence: child.latestEvent.Load()}}
	body, _ := json.Marshal(event)
	page := teamPage{parent: root, compact: true}
	if err := page.prepare(ctx, []storedRecord{{kind: "event", body: body}}); err != nil {
		t.Fatal(err)
	}
	if err := page.close(); err != nil {
		t.Fatal(err)
	}
	page.stores, page.parent = nil, nil
	got, err := page.resolve(ctx, event, true)
	if err != nil || got.Text != "created" || got.Agent != "Hana" {
		t.Fatalf("cached projection: %+v %v", got, err)
	}
}

func TestSharedContextCompactionPreservesOrderAndAuthenticatesOriginals(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	var want []string
	for range 8 {
		value, err := root.AppendTeamMessage(ctx, teamchannel.Message{From: "Hana", Text: strings.Repeat("observed result ", 5000)})
		if err != nil {
			t.Fatal(err)
		}
		message := value.Context()
		want = append(want, message.Content)
		if err := child.CommitContext(ctx, child.session.ID, []runrecord.Entry{{Message: &message}}, value.Sequence); err != nil {
			t.Fatal(err)
		}
	}
	got, _, err := child.MemoryRecords(ctx, child.session.ID, 0, 1<<62)
	if err != nil || len(got) != len(want) {
		t.Fatalf("memory records: %d %v", len(got), err)
	}
	for i, value := range got {
		if value.Role != "peer" || value.Text != want[i] {
			t.Fatal("public source/order changed")
		}
	}
	if _, err := root.db.Exec("DELETE FROM tool_events WHERE kind='team_message'"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := child.MemoryRecords(ctx, child.session.ID, 0, 1<<62); err == nil {
		t.Fatal("compaction accepted a missing public original")
	}
}

func TestSharedContextProjectionAppliesFieldSelectionAndSize(t *testing.T) {
	body := []byte(`{"id":"peer:Hana/1","role":"system","content":null,"reasoning":"omit"}`)
	refs := []byte(`{"[\"content\"]":"body"}`)
	content := strings.Repeat("context", 100)
	raw, _ := json.Marshal(content)
	record := storedRecord{body: body, refs: refs, size: len(body) - 4 + len(raw)}
	message, size, err := (&Store{}).contextRecord(context.Background(), record, map[string]bool{"id": true, "role": true, "content": true}, map[string]string{"body": content})
	if err != nil || message.Content != content || message.Reasoning != "" || size < len(content) {
		t.Fatalf("selected shared projection: %+v %d %v", message, size, err)
	}
	record.body = []byte(`{"role":"system","content":"changed"}`)
	if _, _, err := (&Store{}).contextRecord(context.Background(), record, nil, map[string]string{"body": content}); err == nil {
		t.Fatal("replaced content placeholder accepted")
	}
}
