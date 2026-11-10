// Purpose: Report unowned admissions as uncertain and preserve confirmed native task completion.
// Pattern: Durable task-state projection regression test.
// Usage: go test ./internal/storage/serverrequests -run TestTaskPersistence.
// Related: internal/storage/serverrequests/ledger.go.
package serverrequests

import (
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"testing"
)

func TestTaskPersistenceDistinguishesUnownedAdmissions(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), server.ProjectScope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	task, _, err := l.Admit(t.Context(), "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", "native-request", server.InputDigest("input", nil))
	if err != nil {
		t.Fatal(err)
	}
	value, err := l.Get(t.Context(), task.ID)
	if err != nil || value.Status != catalog.Interrupted || value.Code != catalog.Interrupted || value.Persistence != server.PersistenceUncertain {
		t.Fatal("unowned task presented as confirmed", value, err)
	}
	task.Status = "answered"
	if err := l.Finish(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	value, err = l.Get(t.Context(), task.ID)
	if err != nil || value.Status != "answered" || value.Persistence != server.PersistenceConfirmed {
		t.Fatal(value, err)
	}
	var count int
	if err := l.db.QueryRow("SELECT count(*) FROM requests").Scan(&count); err != nil || count != 1 {
		t.Fatal("inspection repeated admission", count, err)
	}
}
