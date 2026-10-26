// Purpose: Verify compressed payload bounds, concurrency and owned-resource cleanup.
// Pattern: Codec boundary and store round-trip tests.
// Usage: go test -race ./internal/storage/sqlite -run 'TestPayloadCodec|TestPayloadZstandard'.
// Related: internal/storage/sqlite/payload_codec.go, internal/storage/sqlite/payload.go.
package sqlite

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/run"
)

func TestPayloadCodecBoundsAndCleanup(t *testing.T) {
	var codec payloadCodec
	raw := []byte(strings.Repeat("code and reasoning\n", 10000))
	compressed, err := codec.encode(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := codec.decode(compressed, len(raw))
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("round trip failed", err)
	}
	for _, size := range []int{0, len(raw) - 1, len(raw) + 1, maxEventBytes + 1} {
		if _, err := codec.decode(compressed, size); err == nil {
			t.Fatalf("accepted incorrect size %d", size)
		}
	}
	if _, err := codec.decode(compressed[:len(compressed)-1], len(raw)); err == nil {
		t.Fatal("truncated payload accepted")
	}
	if err := codec.close(); err != nil || codec.encoder != nil || codec.decoder != nil || codec.buffer != nil {
		t.Fatal("codec resources retained after close", err)
	}
	if _, err := codec.encode(raw); err == nil {
		t.Fatal("closed codec accepted a write")
	}
	if _, err := codec.decode(compressed, len(raw)); err == nil {
		t.Fatal("closed codec accepted a read")
	}
	if err := codec.close(); err != nil {
		t.Fatal(err)
	}
}

func TestPayloadCodecConcurrentUse(t *testing.T) {
	var codec payloadCodec
	t.Cleanup(func() { codec.close() })
	var workers sync.WaitGroup
	for i := range 8 {
		workers.Go(func() {
			raw := []byte(strings.Repeat(string(rune('a'+i)), 50000))
			compressed, err := codec.encode(raw)
			if err != nil {
				t.Error(err)
				return
			}
			got, err := codec.decode(compressed, len(raw))
			if err != nil || !bytes.Equal(got, raw) {
				t.Error("concurrent round trip failed", err)
			}
		})
	}
	workers.Wait()
}

func TestPayloadZstandardStorageAndReopen(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	text := strings.Repeat("payload line\n", 10000)
	if err := s.RecordEvent(context.Background(), s.Session().ID, run.Event{Kind: "thinking", Thinking: text}); err != nil {
		t.Fatal(err)
	}
	var codec int
	if err := s.db.QueryRow("SELECT codec FROM payloads").Scan(&codec); err != nil || codec != 2 {
		t.Fatal("expected Zstandard payload", codec, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := connect(context.Background(), s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	reopened := &Store{db: db, path: s.path, session: s.session}
	t.Cleanup(func() { reopened.Close() })
	page, _, err := reopened.Timeline(context.Background(), s.Session().ID, 0, 10)
	if err != nil || len(page) != 1 || page[0].Event.Thinking != text {
		t.Fatal("stored payload not restored after reopening", err)
	}
}
