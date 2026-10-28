// Purpose: Avoid copying unchanged command output and release completed capture buffers safely.
// Pattern: Capture boundary regression tests and allocation measurement.
// Usage: go test -race ./internal/tools/local/process -run TestCaptureRevision.
// Related: internal/tools/local/process/output.go, internal/tools/local/process/capture.go.
package process

import (
	"strings"
	"testing"
)

func TestCaptureRevision(t *testing.T) {
	var output cappedOutput
	output.Write([]byte(strings.Repeat("x", 64*1024)))
	first := output.snapshot(0)
	if first.Revision == 0 || len(first.Text) != 64*1024 {
		t.Fatal("first snapshot lost output")
	}
	allocations := testing.AllocsPerRun(100, func() {
		if next := output.snapshot(first.Revision); next.Revision != first.Revision || next.Text != "" {
			t.Fatal("unchanged capture was copied")
		}
	})
	if allocations != 0 {
		t.Fatalf("unchanged capture allocated %.0f times", allocations)
	}
	output.Write([]byte("updated"))
	if next := output.snapshot(first.Revision); next.Revision == first.Revision || !strings.HasSuffix(next.Text, "updated") {
		t.Fatal("changed capture reused stale output")
	}
	output.Write([]byte(strings.Repeat("z", captureHalf*2)))
	output.Write([]byte("wrapped tail"))
	output.release()
	if output.String() != "" {
		t.Fatal("released capture retained output")
	}
}
