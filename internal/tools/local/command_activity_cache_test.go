// Purpose: Reuse command-label masking only while the vault revision is unchanged.
// Pattern: Metadata-only cache invalidation regression and benchmark.
// Usage: go test ./internal/tools/local -run TestCommandActivityMask -bench BenchmarkCommandActivityCached.
// Related: internal/tools/local/commands.go, internal/tools/local/command_live.go.
package local

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCommandActivityMaskTracksVaultRevision(t *testing.T) {
	e, _ := testExecutor(t)
	e.BeginCommands(t.Context(), time.Now().Add(time.Minute))
	defer e.StopCommands()
	id, err := e.commands.Start(t.Context(), e.project, "sleep 60 # private", os.Environ(), "call")
	if err != nil {
		t.Fatal(err)
	}
	version, scans := 0, 0
	e.SetRedactionVersion(func() int { return version })
	e.redact = func(text string) string {
		scans++
		if version > 0 {
			return strings.ReplaceAll(text, "private", "[redacted]")
		}
		return text
	}
	e.CommandActivity()
	e.CommandActivity()
	if scans != 1 {
		t.Fatal("unchanged command label was scanned again", scans)
	}
	version++
	if runs := e.CommandActivity(); scans != 2 || strings.Contains(runs[0].Command, "private") {
		t.Fatal("vault revision reused an unsafe label", scans, runs)
	}
	e.StopCommands()
	if err := e.AcknowledgeCommand(id); err != nil {
		t.Fatal(err)
	}
	if len(e.liveOutputs.values) != 0 {
		t.Fatal("acknowledged command retained metadata cache")
	}
}

func BenchmarkCommandActivityCached(b *testing.B) {
	home, project := b.TempDir(), b.TempDir()
	e, err := NewExecutor(project, home, home+"/.meca", func(s string) string { return s })
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := e.Close(); err != nil {
			b.Error(err)
		}
	})
	e.SetRedactionVersion(func() int { return 0 })
	e.BeginCommands(b.Context(), time.Now().Add(time.Minute))
	for range 4 {
		if _, err := e.commands.Start(b.Context(), project, "sleep 60 # "+strings.Repeat("details ", 8192), os.Environ(), "call"); err != nil {
			b.Fatal(err)
		}
	}
	e.CommandActivity()
	b.ReportAllocs()
	for b.Loop() {
		e.CommandActivity()
	}
}
