// Purpose: Reproduce cross-workload payload admission while a tiny restored response is sent.
// Pattern: concurrency contract regression test.
// Usage: go test ./internal/serverapi/openai.
// Related: internal/serverapi/openai/result.go, internal/processmemory/transient.go, internal/storage/sqlite/commit_memory.go.
package openai

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type resultLedger struct{}

func (resultLedger) Admit(context.Context, string, string, string) (server.Task, bool, error) {
	return server.Task{}, false, nil
}
func (resultLedger) Get(context.Context, string) (server.Task, error) { return server.Task{}, nil }
func (resultLedger) Finish(context.Context, server.Task) error        { return nil }

func TestResultDeliveryDoesNotBlockJournalWrites(t *testing.T) {
	ctx := context.Background()
	service, err := server.New(ctx, server.Options{
		Project: t.TempDir(), Ledger: resultLedger{},
		Open: func(context.Context, string, bool, config.Config) (*server.Conversation, error) { return nil, nil },
		List: func(context.Context, string, int) ([]sessionstate.Conversation, string, error) { return nil, "", nil },
		PrepareModel: func(context.Context, connection.Selection, string, setup.Defaults) (server.PreparedModel, error) {
			return server.PreparedModel{}, nil
		},
		UpdateConfig: func(context.Context, func(*config.Config)) (config.Config, error) { return config.Config{}, nil },
		ReadOutput: func(_ context.Context, _ string, _, _ int64, _ int, reserve func(int) error) (string, error) {
			return "ok", reserve(2)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	catalog, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	store, err := catalog.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	h := &Handler{Service: service}
	sending, releaseSend, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- h.withResult(ctx, nil, server.Response{Session: "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", Through: 1}, func(body string) error {
			t.Logf("sending %d-byte restored response", len(body))
			close(sending)
			<-releaseSend
			return nil
		})
	}()
	defer func() {
		close(releaseSend)
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	<-sending
	deadline, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	release, err := processmemory.AcquirePayload(deadline, 1)
	if err == nil {
		release()
	} else {
		t.Errorf("one byte of unrelated payload work blocked by two-byte result: %v", err)
	}
	flush, stopFlush := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stopFlush()
	if err := store.AppendMessage(flush, store.Session().ID, model.Message{Role: "user", Content: strings.Repeat("x", 32<<10)}); err != nil {
		t.Fatalf("unrelated SQLite journal write also blocked: %v", err)
	}
}
