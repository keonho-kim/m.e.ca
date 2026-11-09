// Purpose: Keep split credentials private while appending large live command output without full rewrites.
// Pattern: Stateful projection regression test.
// Usage: go test -race ./internal/tools/local -run TestIncrementalCapture.
// Related: internal/tools/local/command_incremental.go, internal/redaction/chunks.go.
package local

import (
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	"os"
	"strings"
	"testing"
)

func TestIncrementalCaptureMasksCredentialAcrossDeltas(t *testing.T) {
	e, _ := testExecutor(t)
	vault := &credentials.Vault{}
	if err := vault.Remember("private-api-key"); err != nil {
		t.Fatal(err)
	}
	e.newChunkMask = vault.NewChunkMask
	e.redactionVersion = vault.RedactionVersion
	e.redact = vault.Redact
	var mask func(string, bool) string
	firstText := strings.Repeat("line\n", 12000) + "private-api-"
	first, err := e.projectIncrementalCapture("cmd_incremental", "stdout", liveCapture{}, process.Capture{Revision: 1, Text: firstText, Total: int64(len(firstText))}, &mask)
	if err != nil || first.path == "" {
		t.Fatal(err)
	}
	before, err := os.Stat(first.path)
	if err != nil {
		t.Fatal(err)
	}
	next, err := e.projectIncrementalCapture("cmd_incremental", "stdout", first, process.Capture{Revision: 2, Text: "key\nupdated", Append: true, Offset: first.rawBytes, Total: int64(len(firstText) + 11)}, &mask)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(next.path)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("delta rewrote the entire archive", err)
	}
	body, err := os.ReadFile(next.path)
	if err != nil || strings.Contains(string(body), "private-api-") || !strings.Contains(string(body), "[redacted]") || !strings.HasSuffix(string(body), "updated") {
		t.Fatal("split credential or new output lost", err)
	}
}
