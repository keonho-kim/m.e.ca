// Purpose: Exercise real native browser actions, bounded observations, screenshots and isolation.
// Pattern: Opt-in local HTTP browser integration without user profiles or remote model calls.
// Usage: MECA_BROWSER_NATIVE_AGENT=/absolute/agent-browser MECA_BROWSER_EXECUTABLE=/absolute/chrome go test ./internal/tools/browser -run TestNative.
// Related: internal/tools/browser/session.go, internal/tools/browser/runtime/install.go.
package browser

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

func TestNativeBrowser(t *testing.T) {
	agent, chrome := os.Getenv("MECA_BROWSER_NATIVE_AGENT"), os.Getenv("MECA_BROWSER_EXECUTABLE")
	if agent == "" || chrome == "" {
		t.Skip("set explicit task-owned browser binaries for native verification")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><head><title>Fixture</title></head><body><h1>Browser fixture</h1><input id="entry" aria-label="Entry"><button onclick="document.querySelector('#status').textContent=document.querySelector('#entry').value">Apply</button><p id="status">Ready</p><main>` + strings.Repeat(`<p>Bounded output 😀 paragraph</p>`, 800) + `</main></body></html>`))
	}))
	defer server.Close()
	session := New(t.TempDir(), t.TempDir(), func(s string) string { return strings.ReplaceAll(s, "SECRET", "MASKED") }, nil)
	session.prepare = func(context.Context, string) (runtime.Paths, error) {
		return runtime.Paths{Agent: agent, Browser: chrome}, nil
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := session.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	invoke := func(r Request) Page {
		t.Helper()
		out, err := session.Run(t.Context(), r, session.Current())
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	invoke(Request{Action: "open", URL: server.URL})
	invoke(Request{Action: "fill", Target: "#entry", Value: "--cdp SECRET"})
	invoke(Request{Action: "click", Target: "button"})
	out := invoke(Request{View: "text", Selector: "#status"})
	if !strings.Contains(out.Text, "MASKED") || strings.Contains(out.Text, "SECRET") {
		t.Fatal(out)
	}
	invoke(Request{View: "errors"})
	snapshot := invoke(Request{View: "snapshot"})
	if !strings.Contains(snapshot.Text, "Browser fixture") || !strings.Contains(snapshot.Text, "ref=") {
		t.Fatal(snapshot.Text)
	}
	unchanged := invoke(Request{View: "snapshot", Depth: 5})
	if !unchanged.Unchanged || unchanged.ResultID != snapshot.ResultID || unchanged.Text != "" {
		t.Fatal("unchanged native snapshot was replayed", unchanged)
	}
	invoke(Request{Action: "fill", Target: "#entry", Value: "Second status"})
	invoke(Request{Action: "click", Target: "button"})
	changed := invoke(Request{View: "snapshot"})
	if changed.Unchanged || !strings.Contains(changed.Text, "Second status") || changed.ResultID == snapshot.ResultID {
		t.Fatal("native snapshot failed to reflect changed state", changed)
	}
	large := invoke(Request{View: "text", Selector: "main"})
	if len(large.Text) > 8192 || large.NextOffset == 0 {
		t.Fatal("unbounded or missing next page", large)
	}
	page := invoke(Request{ResultID: large.ResultID, Revision: large.Revision, Offset: large.NextOffset})
	if page.Offset != large.NextOffset || page.Revision != large.Revision {
		t.Fatal(page)
	}
	image := invoke(Request{View: "screenshot"})
	defer image.Close()
	if len(image.Image) < 8 || string(image.Image[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("not a real PNG")
	}
	other := New(t.TempDir(), t.TempDir(), func(s string) string { return s }, nil)
	other.prepare = session.prepare
	if _, err := other.Run(t.Context(), Request{Action: "open", URL: server.URL}, ""); err != nil {
		t.Fatal(err)
	}
	independent, err := other.Run(t.Context(), Request{View: "text", Selector: "#status"}, other.Current())
	if err != nil || independent.Text != "Ready" {
		t.Fatal("browser sessions shared state", independent, err)
	}
	if err := other.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	work, sockets := session.work, session.sockets
	invoke(Request{Action: "close"})
	for _, path := range []string{work, sockets} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("retained browser resources", path, err)
		}
	}
	if _, err := session.Run(t.Context(), Request{ResultID: large.ResultID, Revision: large.Revision}, ""); err == nil {
		t.Fatal("closed result remained readable")
	}
	invoke(Request{Action: "open", URL: server.URL})
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	_, err = session.Run(ctx, Request{Action: "wait", Target: "#never-exists"}, session.Current())
	if !errors.Is(err, context.DeadlineExceeded) || session.work != "" {
		t.Fatal("cancel did not close browser", err, session.work)
	}

}
