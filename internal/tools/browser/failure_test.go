// Purpose: Verify failed storage and failed shutdown retain no false completion or leaked admission.
// Pattern: Filesystem and subprocess failure regression with task-owned fixtures.
// Usage: go test ./internal/tools/browser; set MECA_BROWSER_DISK_TEST_DIR to a small test filesystem for ENOSPC.
// Related: internal/tools/browser/session.go, internal/tools/browser/results.go.
package browser

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/processmemory"
	browserruntime "github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

func TestFailedObservationWriteReleasesMemory(t *testing.T) {
	directory := os.Getenv("MECA_BROWSER_DISK_TEST_DIR")
	s := New("", "", func(text string) string { return text }, nil)
	if directory == "" {
		s.work = filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(s.work, nil, 0600); err != nil {
			t.Fatal(err)
		}
	} else {
		var err error
		s.work, err = os.MkdirTemp(directory, "browser-disk-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(s.work) })
	}
	raw, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", 3<<20)})
	if _, err := s.store(raw, 0); err == nil {
		t.Fatal("storage failure not propagated")
	}
	if len(s.results) != 0 {
		t.Fatal("failed write published a result")
	}
	if directory != "" {
		files, err := os.ReadDir(s.work)
		if err != nil || len(files) != 0 {
			t.Fatal("partial observation retained", files, err)
		}
	}
	lease := processmemory.NewPayloadLease()
	defer lease.Close()
	if err := lease.Resize(int(processmemory.PayloadBytes)); err != nil {
		t.Fatal("failed write leaked memory", err)
	}
}

func TestBrowserFailedExecNeverOccupiesSessionSlot(t *testing.T) {
	slots, err := browserCapacity()
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	missing := filepath.Join(base, "missing-helper")
	for range cap(slots) + 1 {
		s := New("", base, func(text string) string { return text }, nil)
		s.prepare = func(context.Context, string) (browserruntime.Paths, error) {
			return browserruntime.Paths{Agent: missing}, nil
		}
		if _, err := s.Run(t.Context(), Request{Action: "open", URL: "http://localhost"}, ""); err == nil {
			t.Fatal("missing executable accepted")
		}
		if len(slots) != 0 || s.work != "" || s.sockets != "" {
			t.Fatal("failed exec retained a slot or session directories")
		}
	}
}

func TestFailedShutdownCanRetry(t *testing.T) {
	s := New("", "", func(text string) string { return text }, nil)
	s.work, s.sockets = t.TempDir(), t.TempDir()
	s.helperStarted = true
	s.paths.Agent = filepath.Join(t.TempDir(), "missing-helper")
	released := 0
	s.releaseBrowser = func() { released++ }
	if err := s.Close(t.Context()); err == nil {
		t.Fatal("unconfirmed shutdown succeeded")
	}
	if released != 0 || s.work == "" || s.stopped {
		t.Fatal("failed shutdown lost ownership")
	}
	// Once shutdown is confirmed, retrying file cleanup must not launch a new daemon.
	s.stopped = true
	s.releaseBrowser()
	s.releaseBrowser = nil
	if err := s.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if released != 1 || s.work != "" || s.sockets != "" {
		t.Fatal("cleanup did not finish")
	}
}
