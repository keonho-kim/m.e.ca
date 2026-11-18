// Purpose: Verify managed browser update and retirement respect concurrent session ownership.
// Pattern: Deterministic fake Stable endpoint with real private version directories.
// Usage: go test -race ./internal/tools/browser/runtime -run 'TestManagedChrome'.
// Related: internal/tools/browser/runtime/chrome_use.go, internal/tools/browser/runtime/managed.go.
package runtime

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagedChromeReadersDuringUpdate(t *testing.T) {
	fixture := &stableTransport{version: "157.0.8103.78"}
	directory := t.TempDir()
	now := time.Now()
	first, err := managedChrome(t.Context(), &http.Client{Transport: fixture}, directory, "linux", "arm64", now)
	if err != nil {
		t.Fatal(err)
	}
	defer first.close()
	fixture.change("157.0.8103.79", "", false)
	waiting := make(chan struct{})
	done := make(chan error, 1)
	entered := make(chan struct{})
	client := &http.Client{Transport: responseTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == stableEndpoint {
			close(entered)
			select {
			case <-waiting:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
		}
		return fixture.RoundTrip(r)
	})}
	go func() {
		use, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now.Add(25*time.Hour))
		if err == nil && use.path == first.path {
			t.Error("update did not select the verified new version")
		}
		if closeErr := use.close(); closeErr != nil {
			t.Error(closeErr)
		}
		done <- err
	}()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("update did not reach Stable lookup: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	warm, warmErr := managedChrome(ctx, client, directory, "linux", "arm64", now.Add(25*time.Hour))
	cancel()
	close(waiting)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	defer warm.close()
	if warmErr != nil || warm.path != first.path || !executable(warm.path) {
		t.Fatal("available version waited for the update", warm.path, warmErr)
	}
}

func TestManagedChromeRetiresOnlyReleasedTrackedVersions(t *testing.T) {
	fixture := &stableTransport{}
	client := &http.Client{Transport: fixture}
	directory := t.TempDir()
	now := time.Now()
	selectVersion := func(version string, days int) chromeUse {
		t.Helper()
		fixture.change(version, "", false)
		use, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now.Add(time.Duration(days)*25*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = use.close() })
		return use
	}
	first := selectVersion("157.0.8103.78", 0)
	second := selectVersion("157.0.8103.79", 1)
	// An installation that predates the marker cannot be proven inactive.
	secondDir := filepath.Dir(filepath.Dir(second.path))
	if err := os.Remove(filepath.Join(secondDir, leaseAwareMarker)); err != nil {
		t.Fatal(err)
	}
	if err := second.close(); err != nil {
		t.Fatal(err)
	}
	third := selectVersion("157.0.8103.80", 2)
	if !executable(first.path) || !executable(second.path) {
		t.Fatal("protected version was retired")
	}
	if err := first.close(); err != nil {
		t.Fatal(err)
	}
	if err := third.close(); err != nil {
		t.Fatal(err)
	}
	fourth := selectVersion("157.0.8103.81", 3)
	if executable(first.path) || !executable(second.path) || !executable(third.path) || !executable(fourth.path) {
		t.Fatal("released, rollback or untracked version retention differs")
	}
}
