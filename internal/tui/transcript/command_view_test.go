// Purpose: Preserve command text and fresh clocks while reusing sanitized immutable bodies.
// Pattern: Presentation equivalence and credential-invalidation tests.
// Usage: Single and team active-command displays use the same bounded CommandView.
// Related: internal/tui/transcript/command_view.go, internal/tui/transcript/commands.go.
package transcript

import (
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config/credentials"
)

func TestCommandViewReusesBodyAndTracksCredentialVersion(t *testing.T) {
	var v CommandView
	vault := &credentials.Vault{}
	scans := 0
	f := Formatter{RedactionVersion: vault.RedactionVersion, Redact: func(text string) string { scans++; return vault.Redact(text) }}
	run := toolapi.CommandRun{ID: "command", Command: "echo newly-secret\nprintf done", Status: "running"}
	first := v.Render(run, f)
	run.ElapsedMillis = 1000
	second := v.Render(run, f)
	if scans != 1 || first == second || !strings.Contains(second, "01") || !strings.Contains(second, "printf done") {
		t.Fatal("fixed body was rescanned or elapsed time stayed stale", scans, first, second)
	}
	vault.Remember("newly-secret")
	if got := v.Render(run, f); scans != 2 || strings.Contains(got, "newly-secret") || !strings.Contains(got, "[redacted]") {
		t.Fatal("credential revision reused an unsafe projection", got, scans)
	}
	run.Status = "exited"
	if v.Render(run, f) != "" || v.source != "" || v.body != "" {
		t.Fatal("completed command retained presentation references")
	}
}

func TestCommandViewMatchesUncachedOutputAndBoundsRetention(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }, RedactionVersion: func() int { return 0 }}
	for _, text := range []string{"ls -la", "line1\n한글\n", strings.Repeat("long command ", 30000)} {
		var view CommandView
		run := toolapi.CommandRun{Command: text, Status: "running", ElapsedMillis: 2300}
		if got, want := view.Render(run, f), f.CommandActivity([]toolapi.CommandRun{run}); got != want {
			t.Fatal("cached command presentation changed")
		}
		if len(text) > 256*1024 && view.body != "" {
			t.Fatal("oversized projection was retained")
		}
	}
}

func BenchmarkLiveCommandView(b *testing.B) {
	vault := &credentials.Vault{}
	vault.Remember("provider-secret")
	f := Formatter{Redact: vault.Redact, RedactionVersion: vault.RedactionVersion}
	run := toolapi.CommandRun{Command: strings.Repeat("a build command\n", 4096), Status: "running"}
	for _, cached := range []bool{false, true} {
		name := "uncached"
		if cached {
			name = "cached"
		}
		b.Run(name, func(b *testing.B) {
			var view CommandView
			view.Render(run, f)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				run.ElapsedMillis++
				if cached {
					view.Render(run, f)
				} else {
					f.CommandActivity([]toolapi.CommandRun{run})
				}
			}
		})
	}
}
