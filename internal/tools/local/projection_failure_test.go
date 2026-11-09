// Purpose: Probe incremental log state after a recoverable archive admission failure.
// Pattern: Controlled projection failure regression tests.
// Usage: Run go test for this package; failures retain task and storage ownership.
// Related: internal/tools/local/command_incremental.go.
package local

import (
	"errors"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	"os"
	"strings"
	"testing"
)

func TestMaskRetryAfterWriteFailure(t *testing.T) {
	e, _ := testExecutor(t)
	vault := &credentials.Vault{}
	if err := vault.Remember("super-secret"); err != nil {
		t.Fatal(err)
	}
	e.newChunkMask = vault.NewChunkMask
	e.redact = vault.Redact
	e.redactionVersion = vault.RedactionVersion
	var mask func(string, bool) string
	initial := strings.Repeat("x", 40000) + "super-"
	previous, err := e.projectIncrementalCapture("cmd_probe", "stdout", liveCapture{}, process.Capture{Revision: 1, Total: int64(len(initial)), Text: initial}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	update := process.Capture{Revision: 2, Total: int64(len(initial) + len("secret")), Append: true, Offset: int64(len(initial)), Text: "secret"}
	e.retained.limitBytes = int64(previous.bytes + 1)
	if _, err := e.projectIncrementalCapture("cmd_probe", "stdout", previous, update, &mask); err == nil {
		t.Fatal("expected injected capacity failure")
	}
	e.retained.limitBytes = 0
	if _, err := e.projectIncrementalCapture("cmd_probe", "stdout", previous, update, &mask); !errors.Is(err, process.ErrProjectionBusy) {
		t.Fatal("failed projection reused a consumed mask", err)
	}
	next, err := e.projectIncrementalCapture("cmd_probe", "stdout", previous, process.Capture{Revision: 2, Total: update.Total, Text: initial + "secret"}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(next.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(string(body), "secret") || !strings.HasSuffix(string(body), "[redacted]") {
		t.Fatalf("retry exposed credential suffix, final log tail=%q", body[len(body)-16:])
	}
}

func TestMaskRevisionChangeDuringProjection(t *testing.T) {
	e, _ := testExecutor(t)
	vault := &credentials.Vault{}
	e.newChunkMask = vault.NewChunkMask
	e.redactionVersion = vault.RedactionVersion
	e.redact = vault.Redact
	var mask func(string, bool) string
	initial := "future-secret" + strings.Repeat("x", 40000)
	old, err := e.projectIncrementalCapture("cmd_revision", "stdout", liveCapture{}, process.Capture{Revision: 1, Total: int64(len(initial)), Text: initial}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	// This represents a vault change after commandCaptures snapshots its revision,
	// before it applies an already-read append delta.
	if err := vault.Remember("future-secret"); err != nil {
		t.Fatal(err)
	}
	next, err := e.projectIncrementalCapture("cmd_revision", "stdout", old, process.Capture{Revision: 2, Total: int64(len(initial) + 1), Append: true, Offset: int64(len(initial)), Text: "!"}, &mask)
	if errors.Is(err, process.ErrProjectionBusy) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if next.version == vault.RedactionVersion() && strings.Contains(next.excerpt, "future-secret") {
		t.Fatal("old unmasked prefix stamped with the new credential revision")
	}
}

func TestStaleCapturePositionCannotAppendTwice(t *testing.T) {
	e, _ := testExecutor(t)
	vault := &credentials.Vault{}
	e.newChunkMask = vault.NewChunkMask
	e.redactionVersion = vault.RedactionVersion
	var mask func(string, bool) string
	previous, err := e.projectIncrementalCapture("cmd_stale", "stdout", liveCapture{}, process.Capture{Revision: 1, Total: 3, Text: "one"}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	next, err := e.projectIncrementalCapture("cmd_stale", "stdout", previous, process.Capture{Revision: 2, Total: 6, Offset: 3, Text: "two", Append: true}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.projectIncrementalCapture("cmd_stale", "stdout", next, process.Capture{Revision: 3, Total: 11, Offset: 3, Text: "twoagain", Append: true}, &mask); !errors.Is(err, process.ErrProjectionBusy) {
		t.Fatal("stale delta was applied to newer output", err)
	}
}

func TestMaskResetRewritesPreviouslyRetainedLogAfterShrink(t *testing.T) {
	e, _ := testExecutor(t)
	vault := &credentials.Vault{}
	e.newChunkMask = vault.NewChunkMask
	e.redactionVersion = vault.RedactionVersion
	e.redact = vault.Redact
	var mask func(string, bool) string
	key := strings.Repeat("s", 1024)
	original := strings.Repeat(key, 50)
	first, err := e.projectIncrementalCapture("cmd_shrink", "stdout", liveCapture{}, process.Capture{Revision: 1, Total: int64(len(original)), Text: original}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	if err := vault.Remember(key); err != nil {
		t.Fatal(err)
	}
	next, err := e.projectIncrementalCapture("cmd_shrink", "stdout", first, process.Capture{Revision: 2, Total: int64(len(original)), Text: original}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(first.path)
	if err != nil || string(body) != strings.Repeat("[redacted]", 50) || next.path != first.path {
		t.Fatal("reset left the old unmasked retained log", err)
	}
}
