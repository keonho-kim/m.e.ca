// Purpose: Ensure slow ledger reads do not block cancellation and shutdown joins pending model preparation.
// Pattern: Controlled I/O concurrency regression tests.
// Usage: go test -race ./internal/commands/server -run 'TestLedgerRead|TestShutdownDuring'.
// Related: internal/commands/server/operations.go, internal/commands/server/tasks.go.
package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

type waitingLedger struct{ entered, release chan struct{} }

func (l *waitingLedger) Get(ctx context.Context, _ string) (Task, error) {
	close(l.entered)
	select {
	case <-l.release:
		return Task{}, ErrTaskNotFound
	case <-ctx.Done():
		return Task{}, ctx.Err()
	}
}
func (*waitingLedger) Admit(context.Context, string, string, string) (Task, bool, error) {
	return Task{}, false, ErrTaskNotFound
}
func (*waitingLedger) Finish(context.Context, Task) error { return nil }

func TestLedgerReadDoesNotBlockCancellation(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	ledger := &waitingLedger{entered: make(chan struct{}), release: make(chan struct{})}
	s.options.Ledger = ledger
	run, cancel := context.WithCancel(s.ctx)
	defer cancel()
	s.active = &request{task: Task{ID: "active", Status: "running"}, cancel: cancel}
	read := make(chan error, 1)
	go func() { _, err := s.TaskStatus(t.Context(), "older-task"); read <- err }()
	<-ledger.entered
	canceled := make(chan error, 1)
	go func() { canceled <- s.Cancel("active") }()
	ctx, stop := context.WithTimeout(t.Context(), time.Second)
	defer stop()
	select {
	case <-run.Done():
	case <-ctx.Done():
		t.Error("ledger read blocked request cancellation")
	}
	close(ledger.release)
	if err := <-canceled; err != nil {
		t.Fatal(err)
	}
	if err := <-read; !errors.Is(err, ErrTaskNotFound) {
		t.Fatal(err)
	}
}

func TestShutdownDuringModelPreparation(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	entered, prepared := make(chan struct{}), make(chan error, 1)
	s.options.PrepareModel = func(ctx context.Context, _ connection.Selection, _ string, _ setup.Defaults) (PreparedModel, error) {
		close(entered)
		<-ctx.Done()
		return PreparedModel{}, ctx.Err()
	}
	go func() {
		_, err := s.SetModel(t.Context(), ModelSettings{Provider: "ollama", Name: "fixture"})
		prepared <- err
	}()
	<-entered
	if _, err := s.Select(t.Context(), "", true); err == nil {
		t.Fatal("setup did not reserve the session")
	}
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		s.cancel()
		<-closed
		t.Error("shutdown did not cancel and join model preparation")
	}
	if err := <-prepared; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
