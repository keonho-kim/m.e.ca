// Purpose: Verify bounded command capture retains final diagnostics as well as the log beginning.
// Pattern: Output contract tests.
// Usage: go test ./internal/tools/local/process.
// Related: internal/tools/local/process/output.go, internal/tools/local/command.go.
package process

import (
	"strings"
	"testing"
)

func TestLargeOutputPreservesHeadAndFailureTail(t *testing.T) {
	var output cappedOutput
	first := "start\n" + strings.Repeat("progress\n", (captureHalf*2)/9+100)
	if n, err := output.Write([]byte(first)); err != nil || n != len(first) {
		t.Fatalf("write=%d %v", n, err)
	}
	if _, err := output.Write([]byte("file:42 expected ready, got missing\n")); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !output.truncated || !strings.HasPrefix(text, "start\n") || !strings.HasSuffix(text, "file:42 expected ready, got missing\n") || len(text) > captureHalf*2+100 {
		t.Fatalf("invalid capture length=%d truncated=%v", len(text), output.truncated)
	}
}
