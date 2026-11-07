// Purpose: Keep SSE framing ordered while flushing the first frame and bounded event batches.
// Pattern: Deterministic streaming transport regression tests.
// Usage: go test ./internal/serverapi/eventstream -run TestBatch.
// Related: internal/serverapi/eventstream/writer.go, internal/serverapi/eventstream/batch.go.
package eventstream

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (w *flushRecorder) Flush() { w.flushes++; w.ResponseRecorder.Flush() }

func TestBatchPreservesFramesAndFirstDelivery(t *testing.T) {
	out := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	writer, err := New(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Batch(func() error {
		for i := range 3 {
			if err := writer.Send("delta", "cursor", []byte(`{"text":"hello"}`)); err != nil {
				return err
			}
			if out.flushes != 2 {
				t.Fatalf("frame %d: first frame was delayed or following frames were flushed separately: %d", i, out.flushes)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if out.flushes != 3 || out.Body.String() != strings.Repeat("event: delta\nid: cursor\ndata: {\"text\":\"hello\"}\n\n", 3) {
		t.Fatal("batch changed framing or flush count", out.flushes, out.Body.String())
	}
}

func TestBatchFlushesAtTheByteBound(t *testing.T) {
	out := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	writer, err := New(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Send("", "", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Batch(func() error {
		for range 32 {
			if err := writer.Send("delta", "", []byte(strings.Repeat("x", 8192))); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if out.flushes < 6 || out.flushes > 8 {
		t.Fatal("batch did not flush near its 64KiB bound", out.flushes)
	}
	if writer.pending || writer.queued != 0 {
		t.Fatal("batch retained pending application data")
	}
}

func TestBatchDoesNotRetryAfterProducerFailure(t *testing.T) {
	out := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	writer, err := New(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Send("", "", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	want := errors.New("producer failed")
	err = writer.Batch(func() error {
		if err := writer.Send("", "", []byte(`{"before":true}`)); err != nil {
			return err
		}
		return want
	})
	if !errors.Is(err, want) || out.flushes != 2 {
		t.Fatal("failed producer was retried or flushed", err, out.flushes)
	}
	if err := writer.Send("error", "", []byte(`{"error":"failed"}`)); err != nil {
		t.Fatal(err)
	}
	if out.flushes != 3 {
		t.Fatal("batch mode survived failure")
	}
}
