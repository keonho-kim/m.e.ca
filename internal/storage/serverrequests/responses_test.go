// Purpose: Verify atomic compatible request binding and strict result persistence across restart.
// Pattern: SQLite repository contract tests.
// Usage: go test ./internal/storage/serverrequests -run TestResponse.
// Related: internal/storage/serverrequests/responses.go, internal/commands/server/submission.go.
package serverrequests

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/commands/server"
)

func TestResponseReservationIsAtomicBeforeSessionCreation(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	ledger, err := Open(context.Background(), directory, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	scope := strings.Repeat("a", 64)
	value := server.Response{ID: server.ResponseID(scope, "response", "key"), Scope: scope, Digest: strings.Repeat("b", 64), Kind: "response", Stored: true, Status: "queued", Created: 1}
	var wg sync.WaitGroup
	results := make(chan bool, 8)
	for range 8 {
		wg.Go(func() {
			_, duplicate, err := ledger.ReserveResponse(ctx, value)
			if err != nil {
				t.Error(err)
			}
			results <- duplicate
		})
	}
	wg.Wait()
	close(results)
	admitted := 0
	for duplicate := range results {
		if !duplicate {
			admitted++
		}
	}
	if admitted != 1 {
		t.Fatal(admitted)
	}
	value.Digest = strings.Repeat("c", 64)
	if _, _, err := ledger.ReserveResponse(ctx, value); err == nil {
		t.Fatal("different input reused a request key")
	}
	var data []byte
	if err := ledger.db.QueryRow("SELECT response FROM requests LIMIT 1").Scan(&data); err != nil {
		t.Fatal(err)
	}

	if len(data) > 2048 {
		t.Fatal("reservation metadata grew unexpectedly")
	}
	info, err := os.Stat(ledger.path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal(info, err)
	}
}

func TestResponseStorageKeepsIdentityInOnePlace(t *testing.T) {
	l, err := Open(t.Context(), t.TempDir(), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	v := server.Response{ID: server.ResponseID(l.scope, "response", "identity"), Scope: l.scope, Digest: strings.Repeat("b", 64), Kind: "response", Status: "queued", Created: 1}
	if _, _, err := l.ReserveResponse(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	var body []byte
	if err := l.db.QueryRow("SELECT response FROM requests").Scan(&body); err != nil {
		t.Fatal(err)
	}
	for _, identity := range []string{v.ID, v.Scope, v.Digest} {
		if strings.Contains(string(body), identity) {
			t.Fatal("normalized identity was duplicated in metadata")
		}
	}
	v.Session, v.Task, v.Status = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", "task_"+v.ID[5:], "in_progress"
	if err := l.SaveResponse(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	var session, status, message string
	if err := l.db.QueryRow("SELECT session,status,error FROM requests").Scan(&session, &status, &message); err != nil || session != "" || status != "" || message != "" {
		t.Fatal("compatible state was duplicated in native columns", session, status, message, err)
	}
}

func TestResponseCompletionAndStrictRestart(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	ledger, err := Open(context.Background(), directory, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	scope := strings.Repeat("a", 64)
	value := server.Response{ID: server.ResponseID(scope, "response", "key"), Scope: scope, Digest: strings.Repeat("b", 64), Kind: "response", Stored: true, Status: "queued", Created: 1}
	if _, _, err := ledger.ReserveResponse(ctx, value); err != nil {
		t.Fatal(err)
	}
	value.Session = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	value.Task = "task_" + value.ID[5:]
	value.TaskStatus = "stopped"
	value.Status, value.Completed, value.Through = "failed", 2, 17
	value.Error = strings.Repeat("\x01", 4096)
	value.End = work.End{Status: work.Failed, Reason: value.Error}
	if err := ledger.SaveResponse(ctx, value); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	ledger, err = Open(context.Background(), directory, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	read, err := ledger.ReadResponse(ctx, value.ID)
	if err != nil || read.Through != 17 || read.Error != value.Error {
		t.Fatal(read, err)
	}
	value.Previous = server.ResponseID(scope, "response", "changed")
	if err := ledger.SaveResponse(ctx, value); err == nil {
		t.Fatal("completion identity changed")
	}
	var original []byte
	if err := ledger.db.QueryRow("SELECT response FROM requests LIMIT 1").Scan(&original); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.db.Exec("PRAGMA ignore_check_constraints=ON; UPDATE requests SET version=0;"); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.ReadResponse(ctx, value.ID); err == nil {
		t.Fatal("unsupported stored version accepted")
	}
	if _, _, err := ledger.ReserveResponse(ctx, read); err == nil {
		t.Fatal("unsupported row was recreated")
	}
	var after []byte
	if err := ledger.db.QueryRow("SELECT response FROM requests LIMIT 1").Scan(&after); err != nil || string(after) != string(original) {
		t.Fatal("unsupported row changed", err)
	}

	if _, err := ledger.ReadResponse(ctx, "../escape"); err == nil {
		t.Fatal("invalid ID accepted")
	}
	_, err = ledger.ReadResponse(ctx, server.ResponseID(scope, "response", "absent"))
	var public *server.Error
	if !errors.As(err, &public) || public.Code != "not_found" {
		t.Fatal(err)
	}
}

func BenchmarkResponseLifecycle(b *testing.B) {
	l, err := Open(b.Context(), b.TempDir(), strings.Repeat("a", 64))
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	scope := strings.Repeat("a", 64)
	index := 0
	b.ReportAllocs()
	for b.Loop() {
		index++
		value := server.Response{ID: server.ResponseID(scope, "response", strconv.Itoa(index)), Scope: scope, Digest: strings.Repeat("b", 64), Kind: "response", Stored: true, Status: "queued", Created: 1}
		if _, _, err := l.ReserveResponse(b.Context(), value); err != nil {
			b.Fatal(err)
		}
		value.Session = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
		value.Task = "task_" + value.ID[5:]
		value.Status = "in_progress"
		if err := l.SaveResponse(b.Context(), value); err != nil {
			b.Fatal(err)
		}
		value.Status, value.Completed, value.Through = "completed", 2, 1
		value.TaskStatus = "answered"
		value.End = work.End{Status: work.Completed}
		if err := l.SaveResponse(b.Context(), value); err != nil {
			b.Fatal(err)
		}
	}
}
