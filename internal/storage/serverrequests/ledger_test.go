// Purpose: Verify restart-safe admission never repeats a request with uncertain effects.
// Pattern: Private SQLite repository tests.
// Usage: go test ./internal/storage/serverrequests.
// Related: internal/storage/serverrequests/ledger.go.
package serverrequests

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
)

func TestAdmissionSurvivesRestartAndRejectsChangedPayload(t *testing.T) {
	dir := t.TempDir()
	ledger, err := Open(context.Background(), dir, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id := "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	first, duplicate, err := ledger.Admit(ctx, id, "request-1", "change file")
	if err != nil || duplicate {
		t.Fatal(first, duplicate, err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	ledger, err = Open(context.Background(), dir, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	again, duplicate, err := ledger.Admit(ctx, id, "request-1", "change file")
	if err != nil || !duplicate || again.ID != first.ID || again.Status != "interrupted" {
		t.Fatal(again, duplicate, err)
	}
	if _, _, err := ledger.Admit(ctx, id, "request-1", "different change"); err == nil {
		t.Fatal("key reused with different payload")
	}
	first.Status = "answered"
	if err := ledger.Finish(ctx, first); err != nil {
		t.Fatal(err)
	}
	stored, err := ledger.Get(ctx, first.ID)
	if err != nil || stored.Status != "answered" {
		t.Fatal(stored, err)
	}
	if _, err := ledger.Get(ctx, server.TaskID(id, "missing")); !errors.Is(err, server.ErrTaskNotFound) {
		t.Fatal(err)
	}
}

func TestCompletionWithEscapedErrorSurvivesRestart(t *testing.T) {
	directory := t.TempDir()
	ledger, err := Open(context.Background(), directory, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if ledger != nil {
			_ = ledger.Close()
		}
	})
	ctx := t.Context()
	session := "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	task, _, err := ledger.Admit(ctx, session, "request-error", "answer")
	if err != nil {
		t.Fatal(err)
	}
	task.Status = "interrupted"
	// Runtime errors are bounded to 4 KiB before JSON encoding. HTML escaping
	// can expand this valid payload beyond the former 8 KiB read limit.
	task.Error = strings.Repeat("<>&\n", 1024)
	task.Code = "interrupted"
	task.Persistence = server.PersistenceConfirmed
	if err := ledger.Finish(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	ledger, err = Open(context.Background(), directory, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := ledger.Get(ctx, task.ID)
	if err != nil || stored != task {
		t.Fatalf("saved completion could not be restored: %v", err)
	}
	again, duplicate, err := ledger.Admit(ctx, session, "request-error", "answer")
	if err != nil || !duplicate || again != task {
		t.Fatalf("saved completion was not deduplicated: duplicate=%t, error=%v", duplicate, err)
	}
	oversized := task
	oversized.Error = strings.Repeat("<", 4097)
	if err := ledger.Finish(ctx, oversized); err == nil {
		t.Fatal("oversized completion was accepted")
	}
	stored, err = ledger.Get(ctx, task.ID)
	if err != nil || stored != task {
		t.Fatalf("rejected completion changed the stored record: %v", err)
	}
}

func TestNativeCompletionCannotBeRewritten(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	task, _, err := l.Admit(t.Context(), "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", "key", "input")
	if err != nil {
		t.Fatal(err)
	}
	task.Status = "answered"
	if err := l.Finish(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	if err := l.Finish(t.Context(), task); err != nil {
		t.Fatal("identical completion was not idempotent", err)
	}
	task.Status = "interrupted"
	if err := l.Finish(t.Context(), task); err == nil {
		t.Fatal("terminal native task was rewritten")
	}
}

func TestOversizedStoredErrorIsRejectedWithoutChangingOriginal(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	task, _, err := l.Admit(t.Context(), "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", "key", "input")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.db.Exec("UPDATE requests SET error=?", strings.Repeat("x", 1<<20)); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Get(t.Context(), task.ID); err == nil {
		t.Fatal("oversized corrupt error was accepted")
	}
	var size int
	if err := l.db.QueryRow("SELECT length(error) FROM requests").Scan(&size); err != nil || size != 1<<20 {
		t.Fatal("corrupt original was changed", size, err)
	}
}
