// Purpose: Keep bounded journal preparation atomic under overflow and concurrent payload publication.
// Pattern: Controlled transaction interleaving and rollback tests.
// Usage: go test -race ./internal/storage/sqlite -run TestPrepared.
// Related: internal/storage/sqlite/prepare.go, internal/storage/sqlite/prepared_payload.go.
package sqlite

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestPreparedOverflowStillRollsBackTheEntireJournal(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	messages := make([]model.Message, 220)
	entries := make([]record.Entry, len(messages))
	for index := range messages {
		messages[index] = model.Message{Role: "assistant", Content: strings.Repeat("x", 24<<10)}
		entries[index].Message = &messages[index]
	}
	messages[len(messages)-1].Content = "reject"
	prepared, err := s.prepareJournal(t.Context(), entries)
	if err != nil || len(prepared) == 0 || len(prepared) >= len(entries) {
		t.Fatal("fixture did not cross the preparation budget", len(prepared), err)
	}
	if _, err := s.db.ExecContext(t.Context(), `CREATE TRIGGER reject_tail BEFORE INSERT ON messages WHEN NEW.content_bytes=6 BEGIN SELECT RAISE(ABORT,'rejected tail'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(t.Context(), s.Session().ID, entries); err == nil {
		t.Fatal("rejected tail committed")
	}
	var count int
	if err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM messages").Scan(&count); err != nil || count != 0 {
		t.Fatal("prepared prefix escaped rollback", count, err)
	}
	if _, err := s.db.ExecContext(t.Context(), "DROP TRIGGER reject_tail"); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(t.Context(), s.Session().ID, entries); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM messages").Scan(&count); err != nil || count != len(entries) {
		t.Fatal("overflow dropped records", count, err)
	}
}

func TestPreparedKnownPayloadIsRecheckedAtWriteTime(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	message := model.Message{Role: "assistant", Content: strings.Repeat("x", 64<<10)}
	if err := s.AppendMessage(t.Context(), s.Session().ID, message); err != nil {
		t.Fatal(err)
	}
	prepared, err := s.prepareJournal(t.Context(), []record.Entry{{Message: &message}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), "DELETE FROM payloads"); err != nil {
		t.Fatal(err)
	}
	tx, err := s.db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := prepared[0].write(t.Context(), s, tx, s.Session().ID, "now", &journalReceipt{}); err == nil {
		t.Fatal("missing prepared payload produced a dangling reference")
	}
}

func TestPreparedConcurrentPayloadsDeduplicateAtCommit(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	db, err := connect(t.Context(), s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	peer := &Store{db: db, session: s.session}
	defer peer.Close()
	message := model.Message{Role: "assistant", Content: strings.Repeat("shared", 10000)}
	entries := []record.Entry{{Message: &message}}
	first, err := s.prepareJournal(t.Context(), entries)
	if err != nil {
		t.Fatal(err)
	}
	second, err := peer.prepareJournal(t.Context(), entries)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	results := make(chan error, 2)
	for index, store := range []*Store{s, peer} {
		workers.Go(func() {
			prepared := [][]preparedRecord{first, second}[index]
			tx, err := store.db.BeginTx(context.Background(), nil)
			if err != nil {
				results <- err
				return
			}
			defer tx.Rollback()
			if err := prepared[0].write(context.Background(), store, tx, store.Session().ID, "now", &journalReceipt{}); err != nil {
				results <- err
				return
			}
			results <- tx.Commit()
		})
	}
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	var payloads, messages int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM payloads").Scan(&payloads); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&messages); err != nil {
		t.Fatal(err)
	}
	if payloads != 1 || messages != 2 {
		t.Fatal("concurrent publication lost messages or duplicated payloads", payloads, messages)
	}
}
