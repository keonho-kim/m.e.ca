// Purpose: Own one isolated browser lifetime and serialize observations with its actions.
// Pattern: Lazy session adapter with cancellable admission and explicit shutdown.
// Usage: Each local executor owns one Session; request completion and executor close call Close.
// Related: internal/tools/browser/capture.go, internal/tools/browser/results.go.
package browser

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

type Session struct {
	gate                                     chan struct{}
	mu                                       sync.Mutex
	address                                  string
	state, base, work, sockets, config, name string
	paths                                    runtime.Paths
	redact                                   func(string) string
	version                                  func() int
	prepare                                  func(context.Context, string) (runtime.Paths, error)
	results                                  []storedResult
	snapshot                                 snapshotBaseline
	releaseBrowser                           func()
	helperStarted, stopped                   bool
}

func New(state, base string, redact func(string) string, version func() int) *Session {
	return &Session{gate: make(chan struct{}, 1), state: state, base: base, redact: redact, version: version, prepare: runtime.OpenInstalled}
}

// Available reports whether the session's browser dependencies are installed.
func (s *Session) Available() bool { return s != nil && runtime.Available(s.state) }

func (s *Session) Current() string         { s.mu.Lock(); defer s.mu.Unlock(); return s.address }
func (s *Session) setAddress(value string) { s.mu.Lock(); s.address = value; s.mu.Unlock() }

func (s *Session) start(ctx context.Context) (err error) {
	if s.work != "" {
		return nil
	}
	release, err := acquireBrowser()
	if err != nil {
		return err
	}
	started := false
	defer func() {
		if !started {
			release()
		}
	}()
	paths, err := s.prepare(ctx, s.state)
	if err != nil {
		return err
	}
	defer func() {
		if !started {
			err = errors.Join(err, paths.Close())
		}
	}()
	work, err := os.MkdirTemp(s.base, "browser-")
	if err != nil {
		return err
	}
	// Unix socket names must stay short even when macOS TMPDIR has a long prefix.
	sockets, err := os.MkdirTemp("/tmp", "meca-b-")
	if err != nil {
		os.RemoveAll(work)
		return err
	}
	config := filepath.Join(work, "config.json")
	if err := os.WriteFile(config, []byte(`{"json":true,"colorScheme":"light","screenshotFormat":"png"}`), 0600); err != nil {
		os.RemoveAll(work)
		os.RemoveAll(sockets)
		return err
	}
	s.paths, s.work, s.sockets, s.config, s.name = paths, work, sockets, config, "meca-"+rand.Text()
	s.helperStarted = false
	s.stopped = false
	s.releaseBrowser = release
	started = true
	return nil
}

func (s *Session) Run(ctx context.Context, r Request, expectedURL string) (out Page, err error) {
	if err := r.Validate(); err != nil {
		return out, err
	}
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return out, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	fresh := s.work == ""
	defer func() {
		if (ctx.Err() != nil || fresh && err != nil) && s.work != "" {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
			defer cancel()
			err = errors.Join(err, s.closeLocked(cleanup))
		}
	}()
	if r.ResultID != "" {
		return s.page(r)
	}
	if r.Action == "close" {
		return out, s.closeLocked(ctx)
	}
	if r.Action != "open" && (s.work == "" || s.stopped) {
		return out, fault.Errorf(diag.BrowserClosed)
	}
	if s.stopped {
		if err := s.closeLocked(ctx); err != nil {
			return out, err
		}
	}
	if err := s.start(ctx); err != nil {
		return out, err
	}
	if r.View != "" {
		return s.inspect(ctx, r)
	}
	if r.Interaction() {
		address, err := s.url(ctx)
		if err != nil {
			return out, err
		}
		if address != expectedURL {
			return out, fault.Errorf(diag.BrowserStale)
		}
	}
	if r.Action == "open" || r.Action == "reload" || r.Action == "back" || r.Action == "forward" {
		s.snapshot = snapshotBaseline{}
	}
	data, err := s.invoke(ctx, r.args()...)
	if err != nil {
		return out, err
	}

	data.Close()
	if _, err := s.url(ctx); err != nil {
		return out, err
	}
	return Page{URL: s.redact(s.Current()), Text: diag.Text(diag.BrowserActionCompleted)}, nil
}
func (s *Session) url(ctx context.Context) (string, error) {
	data, err := s.invoke(ctx, "get", "url")
	if err != nil {
		return "", err
	}
	defer data.Close()
	address, err := responseURL(data.data)
	if err == nil {
		s.setAddress(address)
	}
	return address, err
}

func (s *Session) Close(ctx context.Context) error {
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	return s.closeLocked(ctx)
}
func (s *Session) closeLocked(ctx context.Context) error {
	if s.work == "" {
		return nil
	}
	if !s.stopped {
		if s.helperStarted {
			receipt, err := s.invoke(ctx, "close")
			if err != nil {
				return fault.Errorf(diag.BrowserClose, err)
			}
			receipt.Close()
		}
		s.stopped = true
		s.results = nil
		s.snapshot = snapshotBaseline{}
		s.setAddress("")
		if s.releaseBrowser != nil {
			s.releaseBrowser()
			s.releaseBrowser = nil
		}
	}
	err := errors.Join(s.paths.Close(), os.RemoveAll(s.work), os.RemoveAll(s.sockets))
	if err != nil {
		return err
	}
	s.work, s.sockets, s.config, s.name = "", "", "", ""
	s.paths = runtime.Paths{}
	return nil
}

func (s *Session) maskVersion() int {
	if s.version == nil {
		return 0
	}
	return s.version()
}
