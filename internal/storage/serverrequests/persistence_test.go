// Purpose: Report unowned native and compatible admissions as uncertain without replaying them.
// Pattern: Durable request-state projection regression tests.
// Usage: go test ./internal/storage/serverrequests -run TestTaskPersistence.
// Related: internal/storage/serverrequests/ledger.go, internal/storage/serverrequests/responses.go.
package serverrequests

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func TestTaskPersistenceDistinguishesUnownedAdmissions(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	session := "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	native, _, err := l.Admit(t.Context(), session, "native-request", "input")
	if err != nil {
		t.Fatal(err)
	}
	response := server.Response{ID: server.ResponseID(l.scope, "response", "compatible-request"), Scope: l.scope, Digest: strings.Repeat("b", 64), Kind: "response", Status: "queued", Created: 1}
	if _, _, err := l.ReserveResponse(t.Context(), response); err != nil {
		t.Fatal(err)
	}
	response.Task, response.Session, response.Status = "task_"+response.ID[5:], session, "in_progress"
	if err := l.SaveResponse(t.Context(), response); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{native.ID, response.Task} {
		value, err := l.Get(t.Context(), id)
		if err != nil || value.Status != catalog.Interrupted || value.Code != catalog.Interrupted || value.Persistence != server.PersistenceUncertain {
			t.Fatal("unowned request was presented as confirmed", value, err)
		}
	}
	native.Status = "answered"
	if err := l.Finish(t.Context(), native); err != nil {
		t.Fatal(err)
	}
	value, err := l.Get(t.Context(), native.ID)
	if err != nil || value.Status != "answered" || value.Persistence != server.PersistenceConfirmed {
		t.Fatal(value, err)
	}
	var count int
	if err := l.db.QueryRow("SELECT count(*) FROM requests").Scan(&count); err != nil || count != 2 {
		t.Fatal("state inspection replayed admission", count, err)
	}
}
