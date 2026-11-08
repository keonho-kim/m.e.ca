// Purpose: Check recoverability after a child admission fails before execution.
// Pattern: Isolated repository behavior probe.
// Usage: go test ./internal/storage/serverrequests.
// Related: internal/storage/serverrequests/responses.go.
package serverrequests

import (
	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/commands/server"
	"testing"
)

func TestFailedChildContinuation(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), server.ProjectScope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	parent := completedResponse(t, l)
	child := server.Response{ID: server.ResponseID(l.scope, "response", "child"), Scope: l.scope, Kind: "response", Digest: l.scope, Previous: parent.ID, Status: "queued", Created: 3, Stored: true}
	if _, _, err := l.ReserveResponse(t.Context(), child); err != nil {
		t.Fatal(err)
	}
	child.Status = "failed"
	child.Completed = 4
	child.End = work.End{Status: work.Failed, Reason: "admission failed before execution"}
	if err := l.SaveResponse(t.Context(), child); err != nil {
		t.Fatal(err)
	}
	retry := server.Response{ID: server.ResponseID(l.scope, "response", "retry"), Scope: l.scope, Kind: "response", Digest: l.scope, Previous: parent.ID, Status: "queued", Created: 5, Stored: true}
	if _, _, err := l.ReserveResponse(t.Context(), retry); err != nil {
		t.Fatalf("parent stays locked by an unexecuted failed child: %v", err)
	}
	if err := l.SaveResponse(t.Context(), child); err != nil {
		t.Fatal(err)
	}
	got, err := l.ReadResponse(t.Context(), parent.ID)
	if err != nil || got.Next != retry.ID {
		t.Fatal("repeated failed-child save cleared another child's claim", got, err)
	}
}

func TestFailedStartedChildKeepsContinuationClaim(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), server.ProjectScope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	parent := completedResponse(t, l)
	child := server.Response{ID: server.ResponseID(l.scope, "response", "started"), Scope: l.scope, Kind: "response", Digest: l.scope, Previous: parent.ID, Status: "queued", Created: 3, Stored: true}
	if _, _, err := l.ReserveResponse(t.Context(), child); err != nil {
		t.Fatal(err)
	}
	child.Task, child.Session, child.Status = "task_"+child.ID[5:], parent.Session, "in_progress"
	if err := l.SaveResponse(t.Context(), child); err != nil {
		t.Fatal(err)
	}
	child.Status, child.Completed, child.TaskStatus = "failed", 4, "interrupted"
	child.End = work.End{Status: work.Failed, Reason: "execution outcome uncertain"}
	if err := l.SaveResponse(t.Context(), child); err != nil {
		t.Fatal(err)
	}
	got, err := l.ReadResponse(t.Context(), parent.ID)
	if err != nil || got.Next != child.ID {
		t.Fatal("uncertain execution claim released", got, err)
	}
	if _, _, err := l.ReserveResponse(t.Context(), server.Response{ID: server.ResponseID(l.scope, "response", "unsafe-retry"), Scope: l.scope, Kind: "response", Digest: l.scope, Previous: parent.ID, Status: "queued", Created: 5, Stored: true}); err == nil {
		t.Fatal("ambiguous child allowed another branch")
	}
}
