// Purpose: Verify installed guide search, actionable failures and cancellation cleanup.
// Pattern: Command integration tests with real shell execution and private fixtures.
// Usage: go test ./internal/commands/develop.
// Related: internal/commands/develop/search.go, internal/config/bootstrap/skills.go, internal/config/statefs/lease_unix.go.
package develop

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/statefs"
)

func TestSearchInstalledReferences(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := Run(t.Context(), home, "delayed task queue", &stdout, &stderr); err != nil {
		t.Fatalf("search: %v\n%s", err, &stderr)
	}
	if !strings.Contains(stdout.String(), "Delayed tasks and expiring entries") {
		t.Fatalf("search missed the installed guide: %s", &stdout)
	}
}

func TestSearchReportsUnavailableOrFailedScript(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	err := Run(t.Context(), home, "queue", &stdout, &stderr)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "meca install") {
		t.Fatalf("missing installation has no repair guidance: %v", err)
	}
	path := filepath.Join(home, ".meca/skills/development/scripts/search.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf 'reference read failed\\n' >&2\nexit 7\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err = Run(t.Context(), home, "queue", &stdout, &stderr)
	var exited *exec.ExitError
	if !errors.As(err, &exited) || exited.ExitCode() != 7 || !strings.Contains(stderr.String(), "reference read failed") {
		t.Fatalf("script failure was lost: %v\n%s", err, &stderr)
	}
}

type readyOutput chan struct{}

func (ready readyOutput) Write(body []byte) (int, error) {
	select {
	case ready <- struct{}{}:
	default:
	}
	return len(body), nil
}

func TestSearchCancellationReleasesInstallationLease(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".meca/skills/development/scripts/search.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nsleep 60 &\nprintf ready\nwait\n"), 0600); err != nil {
		t.Fatal(err)
	}
	deadline, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	ctx, cancel := context.WithCancel(deadline)
	ready := make(readyOutput, 1)
	result := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		result <- Run(ctx, home, "queue", ready, io.Discard)
	}()
	t.Cleanup(func() {
		cancel()
		<-finished
	})
	select {
	case <-ready:
	case err := <-result:
		t.Fatalf("search did not start: %v", err)
	case <-deadline.Done():
		t.Fatal("search did not become ready")
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	lease, err := statefs.Lease(root, true)
	if err == nil {
		lease.Close()
		t.Fatal("installation could replace references during the search")
	}
	if !errors.Is(err, syscall.EWOULDBLOCK) {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation was lost: %v", err)
		}
	case <-deadline.Done():
		t.Fatal("search process group did not stop")
	}
	lease, err = statefs.Lease(root, true)
	if err != nil {
		t.Fatalf("search retained the installation lease: %v", err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
}
