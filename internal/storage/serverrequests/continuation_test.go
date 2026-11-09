// Purpose: Keep continuation heads atomic across independent SQLite owners and reject terminal rewrites.
// Pattern: Concurrent repository boundary tests.
// Usage: go test -race ./internal/storage/serverrequests -run 'TestContinuation|TestTerminal'.
// Related: internal/storage/serverrequests/responses.go, internal/storage/serverrequests/ledger.go.
package serverrequests

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/commands/server"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func completedResponse(t *testing.T, l *Ledger) server.Response {
	t.Helper()
	v := server.Response{ID: server.ResponseID(l.scope, "response", "parent"), Scope: l.scope, Kind: "response", Digest: l.scope, Status: "queued", Created: 1, Stored: true}
	if _, _, err := l.ReserveResponse(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	v.Status, v.Completed, v.Through = "completed", 2, 1
	v.Session, v.Task, v.TaskStatus = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", "task_"+v.ID[5:], "answered"
	v.End = work.End{Status: work.Completed}
	if err := l.SaveResponse(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestContinuationHasOneDurableChild(t *testing.T) {
	directory := t.TempDir()
	l, err := Open(t.Context(), directory, server.ProjectScope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	other, err := Open(t.Context(), directory, l.scope)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	parent := completedResponse(t, l)
	results := make(chan string, 2)
	var writers sync.WaitGroup
	for i, owner := range []*Ledger{l, other} {
		writers.Go(func() {
			v := server.Response{ID: server.ResponseID(l.scope, "response", fmt.Sprint(i)), Scope: l.scope, Kind: "response", Digest: l.scope, Previous: parent.ID, Status: "queued", Created: 3, Stored: true}
			_, _, err := owner.ReserveResponse(t.Context(), v)
			if err == nil {
				results <- v.ID
				return
			}
			var public *faults.Error
			if !errors.As(err, &public) || public.ErrorCode() != "conflict" {
				t.Error(err)
			}
		})
	}
	writers.Wait()
	close(results)
	var children []string
	for id := range results {
		children = append(children, id)
	}
	stored, err := l.ReadResponse(t.Context(), parent.ID)
	var count int
	countErr := l.db.QueryRow("SELECT count(*) FROM requests").Scan(&count)
	if err != nil || countErr != nil || len(children) != 1 || count != 2 || stored.Next != children[0] {
		t.Fatalf("partial or competing continuation: children=%v rows=%d parent=%+v errors=%v,%v", children, count, stored, err, countErr)
	}
}

func TestTerminalResponseCannotReturnToExecution(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), server.ProjectScope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	parent := completedResponse(t, l)
	next := parent
	next.Status, next.TaskStatus, next.Completed = "in_progress", "", 0
	next.End = work.End{}
	if err := l.SaveResponse(t.Context(), next); err == nil {
		t.Fatal("completed request returned to execution")
	}
	stored, err := l.ReadResponse(t.Context(), parent.ID)
	if err != nil || stored.Status != "completed" {
		t.Fatal("rejected update changed terminal state", stored, err)
	}
}
