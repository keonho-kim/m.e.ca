// Purpose: Preserve live logs without rewriting unchanged output or retaining full capture strings.
// Pattern: Filesystem projection regression test.
// Usage: go test ./internal/tools/local -run TestLiveCaptureRevision.
// Related: internal/tools/local/command_live.go, internal/tools/local/process/output.go.
package local

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/tools/local/process"
)

func TestLiveCaptureRevision(t *testing.T) {
	e, _ := testExecutor(t)
	e.redact = func(text string) string { return strings.ReplaceAll(text, "secret", "[redacted]") }
	text := "secret\n" + strings.Repeat("line\n", 20000)
	first, err := e.projectLiveCapture("cmd_test", "stdout", liveCapture{}, process.Capture{Revision: 1, Text: text})
	if err != nil || first.path == "" || len(first.excerpt) > outputExcerptBytes+128 {
		t.Fatalf("invalid first capture: %+v %v", first, err)
	}
	stamp := time.Unix(100, 0)
	if err := os.Chtimes(first.path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	next, err := e.projectLiveCapture("cmd_test", "stdout", first, process.Capture{Revision: 1})
	if err != nil || next != first {
		t.Fatal("unchanged capture lost its excerpt or path")
	}
	info, err := os.Stat(first.path)
	if err != nil || !info.ModTime().Equal(stamp) {
		t.Fatal("unchanged live log was rewritten")
	}
	next, err = e.projectLiveCapture("cmd_test", "stdout", first, process.Capture{Revision: 2, Text: text + "updated"})
	if err != nil || !strings.HasSuffix(next.excerpt, "updated") {
		t.Fatal("updated capture reused an old excerpt")
	}
	body, err := os.ReadFile(next.path)
	if err != nil || strings.Contains(string(body), "secret") || !strings.HasSuffix(string(body), "updated") {
		t.Fatal("live log lost redaction or the newest output")
	}
	e.liveOutputs.command("cmd_test")
	e.liveOutputs.forget("cmd_test")
	if len(e.liveOutputs.values) != 0 {
		t.Fatal("acknowledged command retained its cached excerpt")
	}
}

func TestFinalCaptureReusesPathAndStopsStaleLiveWrites(t *testing.T) {
	e, _ := testExecutor(t)
	id := "cmd_final"
	before := strings.Repeat("early output\n", 4000)
	live, err := e.projectLiveCapture(id, "stdout", liveCapture{}, process.Capture{Revision: 1, Text: before})
	if err != nil {
		t.Fatal(err)
	}
	e.liveOutputs.command(id).streams[0] = live
	final := before + "last output"
	if err := e.archiveCommand(id, "printf final", process.Output{Stdout: final, Status: "exited", Started: true}, nil); err != nil {
		t.Fatal(err)
	}
	observed, err := e.commandCaptures(id)
	if err != nil || observed[0].path != live.path || !strings.HasSuffix(observed[0].excerpt, "last output") {
		t.Fatalf("final capture not reused: %+v %v", observed, err)
	}
	body, err := os.ReadFile(live.path)
	if err != nil || string(body) != final {
		t.Fatal("final log replaced by stale live output", err)
	}
	if files, err := os.ReadDir(e.artifacts); err != nil {
		t.Fatal(err)
	} else {
		logs := 0
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".log") {
				logs++
			}
		}
		if logs != 1 {
			t.Fatalf("command kept %d copies of stdout", logs)
		}
	}
}

func TestFinalCaptureDoesNotRewriteUnchangedMaskedStream(t *testing.T) {
	e, _ := testExecutor(t)
	e.redactionVersion = func() int { return 0 }
	id := "cmd_unchanged"
	text := strings.Repeat("output\n", 8192)
	live, err := e.projectLiveCapture(id, "stdout", liveCapture{}, process.Capture{Revision: 1, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	e.liveOutputs.command(id).streams[0] = live
	stamp := time.Unix(100, 0)
	if err := os.Chtimes(live.path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if err := e.archiveCommand(id, "printf output", process.Output{Stdout: text, Status: "exited", Started: true, Revisions: [2]uint64{1, 0}}, nil); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(live.path)
	if err != nil || !info.ModTime().Equal(stamp) {
		t.Fatal("final archive rewrote an unchanged stream", err)
	}
}
