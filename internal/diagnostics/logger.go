// Purpose: Own bounded asynchronous diagnostic output and join its sole writer on shutdown.
// Pattern: Process-scoped logging I/O boundary.
// Usage: cmd/meca composes Sink and supplies registered reporting through context.
// Related: internal/diagnostics/handler.go, internal/diagnostics/reporter.go, internal/diagnostics/logfile/file.go.
package diagnostics

import (
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxRecordBytes = 4096

var ErrWrite = fault.New(catalog.LogWriteFailed, catalog.DiagnosticWriteFailed)

type Options struct {
	Writer io.Writer
	Level  slog.Level
	Redact func(string) string
	Now    func() time.Time
	// Interrupt must unblock a possibly stalled Writer. A nil callback requires
	// Writer.Write to complete independently; Close always joins the writer.
	Interrupt func() error
}
type repetition struct {
	at         time.Time
	suppressed uint64
}
type Sink struct {
	options       Options
	low, high     chan []byte
	stop, done    chan struct{}
	mu            sync.Mutex
	closed        bool
	recent        map[string]repetition
	dropped       atomic.Uint64
	suppressed    atomic.Uint64
	writeErr      error
	failed        atomic.Bool
	failure       chan struct{}
	outputFailure <-chan struct{}
	once          sync.Once
}

func New(options Options) (*Sink, error) {
	if options.Writer == nil || options.Redact == nil {
		return nil, fault.Errorf(catalog.DiagnosticDependenciesRequired)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	s := &Sink{options: options, low: make(chan []byte, 224), high: make(chan []byte, 32), stop: make(chan struct{}), done: make(chan struct{}), failure: make(chan struct{}), recent: make(map[string]repetition)}
	if output, ok := options.Writer.(interface{ Failure() <-chan struct{} }); ok {
		s.outputFailure = output.Failure()
	}
	go s.run()
	return s, nil
}

func (s *Sink) logger() *slog.Logger { return slog.New(&handler{sink: s}) }
func (s *Sink) Dropped() uint64      { return s.dropped.Load() }

func (s *Sink) accept(level slog.Level, message string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.failed.Load() {
		s.dropped.Add(1)
		return false
	}
	if level >= slog.LevelWarn {
		now := s.options.Now()
		previous, found := s.recent[message]
		if found && now.Sub(previous.at) >= 0 && now.Sub(previous.at) < time.Second {
			previous.suppressed++
			s.recent[message] = previous
			s.suppressed.Add(1)
			return false
		}
		if !found && len(s.recent) >= 64 {
			oldest := ""
			var at time.Time
			for key, value := range s.recent {
				if oldest == "" || value.at.Before(at) {
					oldest, at = key, value.at
				}
			}
			delete(s.recent, oldest)
		}
		s.recent[message] = repetition{at: now}
	}
	return true
}

func (s *Sink) enqueue(level slog.Level, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.failed.Load() {
		s.dropped.Add(1)
		return
	}
	queue := s.low
	if level >= slog.LevelWarn {
		queue = s.high
	}
	select {
	case queue <- body:
	default:
		s.dropped.Add(1)
	}
}

// Close ends acceptance before draining; a record with uncertain partial output is never replayed.
func (s *Sink) Close() error {
	s.once.Do(func() { s.mu.Lock(); s.closed = true; close(s.stop); s.mu.Unlock() })
	if s.options.Interrupt != nil {
		timer := time.NewTimer(catalog.LogDrainTimeout)
		defer timer.Stop()
		select {
		case <-s.done:
		case <-timer.C:
			_ = s.options.Interrupt()
		}
	}
	<-s.done
	return s.writeErr
}

func (s *Sink) Failure() <-chan struct{} { return s.failure }
func (s *Sink) Healthy() bool            { return !s.failed.Load() }
