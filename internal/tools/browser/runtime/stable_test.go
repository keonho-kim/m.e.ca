// Purpose: Verify Stable discovery, verified publication, offline reuse and failed-update rollback.
// Pattern: Deterministic official-URL HTTP fixtures and local executable archives.
// Usage: go test -race ./internal/tools/browser/runtime.
// Related: internal/tools/browser/runtime/stable.go, internal/tools/browser/runtime/managed.go.
package runtime

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type stableTransport struct {
	mu                 sync.Mutex
	version            string
	reported           string
	fail               bool
	metadata, archives int
}

func (f *stableTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	status := 200
	var body []byte
	value, _ := chromeAsset("linux", "arm64", f.version)
	if r.URL.String() == stableEndpoint {
		f.metadata++
		body, _ = json.Marshal(map[string]any{"channels": map[string]any{"Stable": map[string]any{"version": f.version, "downloads": map[string]any{"chrome": []map[string]string{{"platform": "linux-arm64", "url": value.url}}}}}})
	} else if r.URL.String() == value.url {
		f.archives++
		var buffer bytes.Buffer
		archive := zip.NewWriter(&buffer)
		header := &zip.FileHeader{Name: value.executable, Method: zip.Store}
		header.SetMode(0700)
		writer, _ := archive.CreateHeader(header)
		reported := f.reported
		if reported == "" {
			reported = f.version
		}
		fmt.Fprintf(writer, "#!/bin/sh\nprintf 'Google Chrome for Testing %s\\n'\n", reported)
		archive.Close()
		body = buffer.Bytes()
	} else {
		status = 404
	}
	if f.fail {
		status = 503
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: r}, nil
}
func (f *stableTransport) change(version, reported string, fail bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.version, f.reported, f.fail = version, reported, fail
}

func TestManagedStableUpdatesAndRollback(t *testing.T) {
	fixture := &stableTransport{version: "157.0.8103.78"}
	client := &http.Client{Transport: fixture}
	directory := t.TempDir()
	now := time.Now()
	first, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now)
	if err != nil {
		t.Fatal(err)
	}
	fixture.change("157.0.8103.79", "", true)
	cached, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now.Add(time.Hour))
	if err != nil || cached != first || fixture.metadata != 1 {
		t.Fatal("warm cache fetched network", cached, err)
	}
	offline, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now.Add(25*time.Hour))
	if err != nil || offline != first {
		t.Fatal("offline update lost working browser", err)
	}
	fixture.change("157.0.8103.79", "wrong-version", false)
	failed, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now.Add(50*time.Hour))
	if err != nil || failed != first {
		t.Fatal("failed executable replaced working browser", err)
	}
	value, _ := chromeAsset("linux", "arm64", "157.0.8103.79")
	if _, err := os.Stat(filepath.Join(directory, "chrome", "versions", value.name)); !os.IsNotExist(err) {
		t.Fatal("unverified version published", err)
	}
	fixture.change("157.0.8103.79", "", false)
	updated, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now.Add(75*time.Hour))
	if err != nil || updated == first {
		t.Fatal("new stable not selected", updated, err)
	}
	if !executable(first) {
		t.Fatal("running version removed")
	}
	if err := os.WriteFile(filepath.Join(directory, "chrome", "current.json"), []byte(`{"Version":"../../outside"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now); err == nil {
		t.Fatal("corrupt selection silently replaced")
	}
}

func TestConcurrentStableInstall(t *testing.T) {
	fixture := &stableTransport{version: "157.0.8103.78"}
	client := &http.Client{Transport: fixture}
	directory := t.TempDir()
	now := time.Now()
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			if _, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if fixture.metadata != 1 || fixture.archives != 1 {
		t.Fatal("duplicate acquisition", fixture.metadata, fixture.archives)
	}
	entries, err := os.ReadDir(filepath.Join(directory, "chrome", "versions"))
	if err != nil || len(entries) != 1 {
		t.Fatal("partial stages retained", entries, err)
	}
}

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestStableMetadataRejectsUntrustedSource(t *testing.T) {
	for _, body := range []string{
		`{"channels":{"Stable":{"version":"../1","downloads":{"chrome":[]}}}}`,
		`{"channels":{"Stable":{"version":"157.0.8103.78","downloads":{"chrome":[{"url":"https://evil.example/chrome.zip"}]}}}}`,
		`{"channels":{"Beta":{"version":"157.0.8103.78"}}}`, strings.Repeat("x", (1<<20)+1),
	} {
		client := &http.Client{Transport: responseTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
		})}
		if _, err := latestStable(t.Context(), client, "linux", "arm64"); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}

func TestChromeRedirectCannotChangeSource(t *testing.T) {
	origin, _ := url.Parse("https://storage.googleapis.com/chrome-for-testing-public/157.0.8103.78/linux-arm64/chrome-linux-arm64.zip")
	for _, target := range []string{"https://evil.example/archive.zip", "http://storage.googleapis.com/archive.zip", origin.String() + "?alternate=1", "https://storage.googleapis.com/another-bucket/archive.zip"} {
		redirected, _ := url.Parse(target)
		if err := runtimeRedirect(&http.Request{URL: redirected}, []*http.Request{{URL: origin}}); err == nil {
			t.Fatal("redirect escaped official archive", target)
		}
	}
}
