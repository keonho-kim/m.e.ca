// Purpose: Verify repeated native opens and browser crashes release admission and temporary resources.
// Pattern: Opt-in isolated process lifecycle integration.
// Usage: Set explicit native binary paths and run TestNativeBrowserLifetime.
// Related: internal/tools/browser/session.go, internal/tools/browser/native_test.go.
package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	browserruntime "github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

func TestNativeBrowserLifetime(t *testing.T) {
	agent, chrome := os.Getenv("MECA_BROWSER_NATIVE_AGENT"), os.Getenv("MECA_BROWSER_EXECUTABLE")
	if agent == "" || chrome == "" {
		t.Skip("explicit native binary paths required")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>Lifetime fixture</body></html>"))
	}))
	defer server.Close()
	s := New(t.TempDir(), t.TempDir(), func(text string) string { return text }, nil, nil)
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
	for cycle := range 8 {
		if _, err := s.Run(t.Context(), Request{Action: "open", URL: server.URL}, ""); err != nil {
			t.Fatal(err)
		}
		work, sockets := s.work, s.sockets
		if cycle == 3 {
			root, err := os.ReadFile(filepath.Join(s.sockets, s.name+".pid"))
			if err != nil {
				t.Fatal(err)
			}
			output, err := exec.CommandContext(t.Context(), "ps", "-A", "-o", "pid=,ppid=,comm=").Output()
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, row := range strings.Split(string(output), "\n") {
				fields := strings.Fields(row)
				if len(fields) < 3 || fields[1] != strings.TrimSpace(string(root)) {
					continue
				}
				if !strings.Contains(strings.ToLower(row), "chrom") {
					continue
				}
				pid, err := strconv.Atoi(fields[0])
				if err != nil {
					t.Fatal(err)
				}
				child, err := os.FindProcess(pid)
				if err != nil {
					t.Fatal(err)
				}
				err = child.Kill()
				child.Release()
				if err != nil {
					t.Fatal(err)
				}
				found = true
			}
			if !found {
				t.Fatal("owned Chrome child not found")
			}
		}
		if err := s.Close(t.Context()); err != nil {
			t.Fatal("close after cycle", cycle, err)
		}
		for _, path := range []string{work, sockets} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("retained resources", path, err)
			}
		}
	}
	slots, err := browserCapacity()
	if err != nil || len(slots) != 0 {
		t.Fatal("lifecycle leaked admission", err)
	}
}
