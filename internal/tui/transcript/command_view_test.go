// Purpose: Preserve command text and fresh clocks while reusing sanitized immutable bodies.
// Pattern: Presentation equivalence and credential-invalidation tests.
// Usage: Single-mode active-command displays use this bounded CommandView.
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
	first := v.Render(run, f, false)
	run.ElapsedMillis = 1000
	second := v.Render(run, f, false)
	if scans != 1 || first == second || !strings.Contains(second, "01") || !strings.Contains(second, "printf done") {
		t.Fatal("fixed body was rescanned or elapsed time stayed stale", scans, first, second)
	}
	vault.Remember("newly-secret")
	if got := v.Render(run, f, false); scans != 2 || strings.Contains(got, "newly-secret") || !strings.Contains(got, "[redacted]") {
		t.Fatal("credential revision reused an unsafe projection", got, scans)
	}
	run.Status = "exited"
	if shown := v.Render(run, f, false); shown == "" || v.source != "" || v.body != "" {
		t.Fatal("completed command retained presentation references")
	}
}

func TestCommandViewMatchesUncachedOutputAndBoundsRetention(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }, RedactionVersion: func() int { return 0 }}
	for _, text := range []string{"ls -la", "line1\n한글\n", strings.Repeat("long command ", 30000)} {
		var view CommandView
		run := toolapi.CommandRun{Command: text, Status: "running", ElapsedMillis: 2300}
		if got, want := view.Render(run, f, false), f.CommandActivity([]toolapi.CommandRun{run}); got != want {
			t.Fatal("cached command presentation changed")
		}
		if len(text) > 256*1024 && view.body != "" {
			t.Fatal("oversized projection was retained")
		}
	}
}

func TestCommandViewFoldChangesRefreshBodyWithoutRescanningEachTick(t *testing.T) {
	var view CommandView
	vault := &credentials.Vault{}
	scans := 0
	f := Formatter{RedactionVersion: vault.RedactionVersion, Redact: func(text string) string { scans++; return vault.Redact(text) }}
	run := toolapi.CommandRun{Command: "echo newly-secret\n" + strings.Repeat("command body\n", 30) + "COMMAND_TAIL", Status: "running"}
	for _, collapsed := range []bool{true, false, true} {
		before := scans
		shown := view.Render(run, f, collapsed)
		if strings.Contains(shown, "COMMAND_TAIL") == collapsed {
			t.Fatal("command view retained the previous fold representation")
		}
		run.ElapsedMillis += 1000
		view.Render(run, f, collapsed)
		if scans != before+1 {
			t.Fatal("unchanged command was rescanned on the next clock tick")
		}
	}
	vault.Remember("newly-secret")
	if shown := view.Render(run, f, true); strings.Contains(shown, "newly-secret") || !strings.Contains(shown, "[redacted]") {
		t.Fatal("folded body ignored the current credential revision")
	}
	if shown := view.Render(run, f, false); strings.Contains(shown, "newly-secret") || !strings.Contains(shown, "COMMAND_TAIL") {
		t.Fatal("expansion lost the command or exposed an older mask")
	}
	run.Status = "exited"
	if shown := view.Render(run, f, true); strings.Contains(shown, "COMMAND_TAIL") || view.source != "" || view.body != "" {
		t.Fatal("terminal snapshot ignored folding or retained its source")
	}
}

func TestCommandViewWithoutRevisionDoesNotReuseStaleText(t *testing.T) {
	var view CommandView
	vault := &credentials.Vault{}
	f := Formatter{Redact: vault.Redact}
	run := toolapi.CommandRun{Command: "echo newly-secret", Status: "running"}
	view.Render(run, f, true)
	vault.Remember("newly-secret")
	if shown := view.Render(run, f, true); strings.Contains(shown, "newly-secret") || view.body != "" {
		t.Fatal("unversioned redaction reused a stale retained body")
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
			view.Render(run, f, false)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				run.ElapsedMillis++
				if cached {
					view.Render(run, f, false)
				} else {
					f.CommandActivity([]toolapi.CommandRun{run})
				}
			}
		})
	}
}

func BenchmarkFoldedCommandView(b *testing.B) {
	vault := &credentials.Vault{}
	f := Formatter{Redact: vault.Redact, RedactionVersion: vault.RedactionVersion}
	run := toolapi.CommandRun{Command: strings.Repeat("a build command\n", 4096), Status: "running"}
	var view CommandView
	view.Render(run, f, true)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		run.ElapsedMillis++
		view.Render(run, f, true)
	}
}
