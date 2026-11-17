// Purpose: Verify oversized native browser output is rejected while browser shutdown remains available.
// Pattern: Opt-in native integration of capture admission and cleanup.
// Usage: Set MECA_BROWSER_NATIVE_AGENT and MECA_BROWSER_EXECUTABLE, then run TestNativeBrowserCaptureLimit.
// Related: internal/tools/browser/capture.go, internal/tools/browser/capture_test.go.
package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
	browserruntime "github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

func TestNativeBrowserCaptureLimit(t *testing.T) {
	agent, chrome := os.Getenv("MECA_BROWSER_NATIVE_AGENT"), os.Getenv("MECA_BROWSER_EXECUTABLE")
	if agent == "" || chrome == "" {
		t.Skip("opt-in capture integration with explicit native binary paths")
	}
	body := `<html><body><pre>` + strings.Repeat("bounded line\n", maxCapture/13+2048) + `</pre></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	s := New(t.TempDir(), t.TempDir(), func(text string) string { return text }, nil)
	s.prepare = func(context.Context, string) (browserruntime.Paths, error) {
		return browserruntime.Paths{Agent: agent, Browser: chrome}, nil
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	if _, err := s.Run(t.Context(), Request{Action: "open", URL: server.URL}, ""); err != nil {
		t.Fatal(err)
	}
	_, err := s.Run(t.Context(), Request{View: "text", Selector: "pre"}, s.Current())
	if fault.Code(err) != string(diag.ResourceExhausted) {
		t.Fatal("oversized output bypassed capture admission", err)
	}
	if len(s.results) != 0 {
		t.Fatal("oversized result retained")
	}
	work, sockets := s.work, s.sockets
	blocker := processmemory.NewPayloadLease()
	defer blocker.Close()
	if err := blocker.Resize(int(processmemory.PayloadBytes)); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(t.Context()); err != nil {
		t.Fatal("memory saturation prevented shutdown", err)
	}
	for _, path := range []string{work, sockets} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("capture failure retained artifacts", path, err)
		}
	}
}
