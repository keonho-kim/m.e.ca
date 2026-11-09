// Purpose: Observe output failure immediately and interrupt/join stalled owned writers.
// Pattern: Deterministic diagnostic ownership contract tests.
// Usage: go test -race ./internal/diagnostics -run 'TestDiagnostic|TestWarningKeys'.
// Related: internal/diagnostics/logger.go, internal/diagnostics/writer.go.
package diagnostics

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"io"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"testing/synctest"
	"time"
)

func TestDiagnosticSummaryReportsMissingContext(t *testing.T) {
	before := missingReporters.Load()
	Log(context.Background(), catalog.RunStarted)
	var output bytes.Buffer
	sink, err := New(Options{Writer: &output, Redact: func(v string) string { return v }})
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Missing uint64 `json:"missing_reporters"`
	}
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil || summary.Missing < before+1 {
		t.Fatal("unconfigured reporter was not observable", summary, err)
	}
}

func TestDiagnosticFailureSignal(t *testing.T) {
	sink, err := New(Options{Writer: failingWriter{}, Redact: func(v string) string { return v }})
	if err != nil {
		t.Fatal(err)
	}
	sink.logger().Error("failed")
	<-sink.Failure()
	if sink.Healthy() {
		t.Fatal("failed output still reported healthy")
	}
	if err := sink.Close(); err != ErrWrite {
		t.Fatal(err)
	}
}

type partialBusyWriter struct{}

func (partialBusyWriter) Write([]byte) (int, error) { return 1, syscall.EAGAIN }

func TestDiagnosticPartialBackpressureEndsOutput(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sink, err := New(Options{Writer: partialBusyWriter{}, Redact: func(v string) string { return v }})
		if err != nil {
			t.Fatal(err)
		}
		defer sink.Close()
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		sink.logger().Info("started")
		select {
		case <-sink.Failure():
		case <-ctx.Done():
			t.Fatal("partially written record was treated as safe backpressure")
		}
	})
}

type queuedLogWriter struct {
	entered, release chan struct{}
	once             sync.Once
	mu               sync.Mutex
	records          []string
}

func (w *queuedLogWriter) Write(body []byte) (int, error) {
	w.once.Do(func() { close(w.entered); <-w.release })
	w.mu.Lock()
	w.records = append(w.records, string(body))
	w.mu.Unlock()
	return len(body), nil
}

func TestDiagnosticSummarySurvivesBusyPriorityQueues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		writer := &queuedLogWriter{entered: make(chan struct{}), release: make(chan struct{})}
		unblock := sync.OnceFunc(func() { close(writer.release) })
		defer unblock()
		sink, err := New(Options{Writer: writer, Redact: func(v string) string { return v }})
		if err != nil {
			t.Fatal(err)
		}
		defer sink.Close()
		logger := sink.logger()
		logger.Warn("first")
		<-writer.entered
		for range 300 {
			logger.Info("queued")
		}
		for i := range 24 {
			logger.Warn("queued-high-" + strconv.Itoa(i))
		}
		time.Sleep(catalog.LogSummaryInterval)
		unblock()
		if err := sink.Close(); err != nil {
			t.Fatal(err)
		}
		if len(writer.records) < 2 || !strings.Contains(writer.records[1], string(catalog.ReporterSummary)) {
			t.Fatal("periodic loss summary starved behind the priority queues")
		}
	})
}

func TestDiagnosticCloseInterruptsAndJoinsWriter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		writer := &blockedWriter{entered: make(chan struct{}), release: make(chan struct{})}
		interrupt := sync.OnceFunc(func() { close(writer.release) })
		sink, err := New(Options{Writer: writer, Redact: func(v string) string { return v }, Interrupt: func() error { interrupt(); return nil }})
		if err != nil {
			t.Fatal(err)
		}
		defer interrupt()
		sink.logger().Info("started")
		<-writer.entered
		done := make(chan error, 1)
		go func() { done <- sink.Close() }()
		time.Sleep(catalog.LogDrainTimeout)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		select {
		case <-sink.done:
		default:
			t.Fatal("writer was not joined")
		}
	})
}

func TestWarningKeysRemainBoundedAndSuppressNewKeys(t *testing.T) {
	sink, err := New(Options{Writer: io.Discard, Now: func() time.Time { return time.Unix(1000, 0) }, Redact: func(v string) string { return v }})
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	for i := range 80 {
		sink.logger().Warn("failure", "error_code", strconv.Itoa(i))
	}
	before := sink.suppressed.Load()
	sink.logger().Warn("failure", "error_code", "79")
	if sink.suppressed.Load() != before+1 || len(sink.recent) > 64 {
		t.Fatal("new warning keys bypassed bounded suppression")
	}
}
