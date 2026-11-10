// Purpose: Reproduce codec/connection lock inversion in streamed previews and in-transaction preparation.
// Pattern: Controlled I/O interleaving regression test.
// Usage: Run go test for this package; failures retain task and storage ownership.
// Related: internal/storage/sqlite/commit.go, internal/storage/sqlite/text_prefix.go.
package sqlite

import (
	"context"
	"encoding/hex"
	"errors"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/content"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type auditPausedReader struct {
	once             sync.Once
	entered, release chan struct{}
	source           io.Reader
}

func (r *auditPausedReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.entered); <-r.release })
	return r.source.Read(p)
}

func TestPreviewAndTransactionLockOrder(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AppendMessage(t.Context(), s.Session().ID, modelapi.Message{Role: "assistant", Content: strings.Repeat("a", 1<<20)}); err != nil {
		t.Fatal(err)
	}
	var id string
	var codec, size, stored int
	var digest []byte
	if err := s.db.QueryRow("SELECT id,codec,raw_bytes,length(body),digest FROM payloads LIMIT 1").Scan(&id, &codec, &size, &stored, &digest); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	reader := &auditPausedReader{entered: make(chan struct{}), release: make(chan struct{}), source: &payloadChunks{ctx: ctx, store: s, id: id, size: stored}}
	readDone := make(chan error, 1)
	go func() {
		_, _, err := content.StreamTextPrefix(ctx, codec, size, hex.EncodeToString(digest), reader, 128, false, nil)
		readDone <- err
	}()
	<-reader.entered // Pause before SQLite I/O so an in-transaction encode can run concurrently.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	writeDone := make(chan error, 1)
	go func() {
		encoder := journalEncoder{store: s, ctx: ctx, lookup: tx}
		_, err := encoder.prepare(record.Entry{Message: &modelapi.Message{Role: "assistant", Content: strings.Repeat("b", 1<<20)}})
		_ = tx.Rollback()
		writeDone <- err
	}()
	close(reader.release)
	readErr, writeErr := <-readDone, <-writeDone
	if errors.Is(readErr, context.DeadlineExceeded) || ctx.Err() != nil {
		t.Fatalf("preview and in-tx preparation only broke lock cycle at cancellation: read=%v write=%v", readErr, writeErr)
	}
}
