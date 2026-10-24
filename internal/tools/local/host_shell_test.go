//go:build linux || darwin

// Purpose: Verify approved host commands retain cache access and accept existing hardlinks.
// Pattern: Native execution regression tests.
// Usage: go test ./internal/tools/local -run TestHostShell.
// Related: internal/tools/local/command.go, internal/tools/local/process/run_unix.go.
package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestHostShellCacheAndHardlinks(t *testing.T) {
	for _, linked := range []bool{false, true} {
		project, home := t.TempDir(), t.TempDir()
		t.Setenv("HOME", home)
		executor, err := NewExecutor(project, home, filepath.Join(home, ".meca"), func(s string) string { return s })
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := executor.Close(); err != nil {
				t.Error(err)
			}
		})
		if linked {
			original := filepath.Join(home, "cached-package")
			if err := os.WriteFile(original, []byte("package"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Link(original, filepath.Join(project, "installed-package")); err != nil {
				t.Fatal(err)
			}
		}
		op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"mkdir -p \"$HOME/.cache/tool\" && printf ready > \"$HOME/.cache/tool/lock\" && ls . && cat \"$HOME/.cache/tool/lock\""}`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := op.Execute(context.Background(), false); err == nil {
			t.Fatal("unapproved host command ran")
		}
		result, err := op.Execute(context.Background(), true)
		if err != nil || result.Failed || !result.Started || result.Process.ExitCode != 0 {
			t.Fatalf("linked=%t result=%+v err=%v", linked, result, err)
		}
		if data, err := os.ReadFile(filepath.Join(home, ".cache/tool/lock")); err != nil || string(data) != "ready" {
			t.Fatalf("cache unavailable: %q %v", data, err)
		}
		var output struct{ Stdout string }
		if err := json.Unmarshal(result.Content, &output); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.Stdout, "ready") {
			t.Fatal("host output missing")
		}
	}
}
