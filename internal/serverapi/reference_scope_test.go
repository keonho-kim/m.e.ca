// Purpose: Deliver reference invalidation to retained conversations while keeping their message bodies scoped.
// Pattern: Public stream filtering contract regression.
// Usage: go test ./internal/serverapi -run TestReferenceChangesRemainGlobal.
// Related: internal/serverapi/watch_scope.go, webui/src/pages/workspace/api/event-cache.ts.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	"testing"
)

func TestReferenceChangesRemainGlobal(t *testing.T) {
	scope := bodyScope{workspace: "selected", session: "one", scoped: true, sessionOnly: true}
	change := server.Event{Workspace: "other", Session: "two", MutationKey: "reference-change", Record: record.Event{Kind: "file_references_changed"}}
	body := server.Event{Workspace: "other", Session: "two", Record: record.Event{Kind: "assistant_delta", Text: "private body"}}
	got := scope.filter(server.Batch{Events: []server.Event{change, body}})
	if len(got.Events) != 1 || got.Events[0].MutationKey != change.MutationKey {
		t.Fatalf("reference invalidation was lost or body leaked: %+v", got.Events)
	}
}
