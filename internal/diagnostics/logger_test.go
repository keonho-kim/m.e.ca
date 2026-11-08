// Purpose: Verify safe structured diagnostics, bounded slow-output queues and joined shutdown.
// Pattern: Deterministic logging boundary tests.
// Usage: go test ./internal/diagnostics.
// Related: internal/diagnostics/logger.go, internal/diagnostics/handler.go.
package diagnostics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestSafeStructuredFieldsAndContextRedaction(t *testing.T) {
	var output bytes.Buffer
	sink, err := New(Options{Writer: &output, Level: slog.LevelDebug, Redact: func(s string) string { return s }})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRedactor(withLogger(context.Background(), sink.logger()), func(s string) string { return strings.ReplaceAll(s, "canary-credential", "[redacted]") })
	logger(ctx).InfoContext(ctx, "request_finished", "task_id", "canary-credential", "authorization", "never-store-this", "status", "answered")
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "canary-credential") || strings.Contains(output.String(), "never-store-this") {
		t.Fatal(output.String())
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
		t.Fatal(err)
	}
	if record["task_id"] != "[redacted]" || record["status"] != "answered" {
		t.Fatal(record)
	}
}

type blockedWriter struct {
	entered, release chan struct{}
	once             sync.Once
}

func (w *blockedWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	<-w.release
	return len(p), nil
}

func TestSlowWriterDoesNotBlockExecutionAndQueueIsBounded(t *testing.T) {
	w := &blockedWriter{entered: make(chan struct{}), release: make(chan struct{})}
	sink, err := New(Options{Writer: w, Redact: func(s string) string { return s }})
	if err != nil {
		t.Fatal(err)
	}
	sink.logger().Info("started")
	<-w.entered
	for range 2000 {
		sink.logger().Info("event", "task_id", strings.Repeat("x", 1000))
	}
	if sink.Dropped() == 0 {
		t.Fatal("unbounded queue or no overflow accounting")
	}
	close(w.release)
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("secret backend error") }

func TestWriterFailureIsReportedWithoutLeakingBackendText(t *testing.T) {
	sink, err := New(Options{Writer: failingWriter{}, Redact: func(s string) string { return s }})
	if err != nil {
		t.Fatal(err)
	}
	sink.logger().Error("failure")
	if err := sink.Close(); !errors.Is(err, ErrWrite) || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
}

func TestRedactionPrecedesFieldClipping(t *testing.T) {
	var output bytes.Buffer
	secret := strings.Repeat("private-canary-", 50)
	sink, err := New(Options{Writer: &output, Redact: func(s string) string { return strings.ReplaceAll(s, secret, "[redacted]") }})
	if err != nil {
		t.Fatal(err)
	}
	sink.logger().Info("metadata", "task_id", secret)
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "private-canary") {
		t.Fatal("clipping exposed a credential prefix")
	}
}

func TestScopedAttributesPreserveSlogGroups(t *testing.T) {
	var output bytes.Buffer
	sink, err := New(Options{Writer: &output, Redact: func(s string) string { return s }})
	if err != nil {
		t.Fatal(err)
	}
	sink.logger().With("component", "engine").WithGroup("request").With("task_id", "task_one").Info("event", "status", "answered")
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &fields); err != nil {
		t.Fatal(err)
	}
	group, ok := fields["request"].(map[string]any)
	if !ok || group["task_id"] != "task_one" || group["status"] != "answered" || fields["component"] != "engine" {
		t.Fatal(fields)
	}
}

func BenchmarkLogMetadata(b *testing.B) {
	sink, err := New(Options{Writer: io.Discard, Redact: func(s string) string { return s }})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = sink.Close() })
	logger := sink.logger()
	b.ReportAllocs()
	for b.Loop() {
		logger.Info("request_finished", "task_id", "task_fixture", "status", "answered")
	}
}
