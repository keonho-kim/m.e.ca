// Purpose: Verify lazy runtime publication, integrity failures, shared locking and cancellation.
// Pattern: Deterministic local HTTP and filesystem contract tests.
// Usage: go test -race ./internal/tools/browser/runtime.
// Related: internal/tools/browser/runtime/install.go, internal/tools/browser/runtime/download.go.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentInstallAndTamper(t *testing.T) {
	body := []byte("native fixture executable")
	digest := sha256.Sum256(body)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write(body) }))
	defer server.Close()
	a := asset{name: "pinned-helper", url: server.URL, digest: hex.EncodeToString(digest[:]), executable: "agent-browser"}
	parent := t.TempDir()
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			path, err := install(t.Context(), server.Client(), parent, a)
			if err != nil {
				t.Error(err)
			} else if !executable(path) {
				t.Error("not executable")
			}
		})
	}
	workers.Wait()
	if calls.Load() != 1 {
		t.Fatal("duplicate downloads", calls.Load())
	}
	path := filepath.Join(parent, a.name, a.executable)
	if err := os.WriteFile(path, []byte("changed"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := install(t.Context(), server.Client(), parent, a); err == nil {
		t.Fatal("tampered cache accepted")
	}
	if calls.Load() != 1 {
		t.Fatal("corrupt data was replaced silently")
	}
}
func TestFailedDownloadNeverPublishes(t *testing.T) {
	for _, status := range []int{200, 404, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); w.Write([]byte("wrong digest")) }))
			defer server.Close()
			parent := t.TempDir()
			_, err := install(t.Context(), server.Client(), parent, asset{name: "helper", url: server.URL, digest: "unmatched", executable: "agent-browser"})
			if err == nil {
				t.Fatal("failed acquisition succeeded")
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 0 {
				t.Fatal("retained partial installation", entries, err)
			}
		})
	}
}
func TestCancellationRemovesStage(t *testing.T) {
	entered := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	parent := t.TempDir()
	done := make(chan error, 1)
	go func() {
		_, err := install(ctx, server.Client(), parent, asset{name: "helper", url: server.URL})
		done <- err
	}()
	<-entered
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
}
func TestPlatformSelection(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		for _, arch := range []string{"amd64", "arm64"} {
			a, err := agentAsset(goos, arch, false)
			if err != nil || len(a.digest) != 64 {
				t.Fatal(a, err)
			}
		}
	}
	if a, err := chromeAsset("linux", "arm64", "157.0.8103.78"); err != nil || a.digest != "" || !a.zip {
		t.Fatal(a, err)
	}
}
