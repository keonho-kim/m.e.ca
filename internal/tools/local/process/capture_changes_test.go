// Purpose: Verify changed-byte projection avoids copying retained prefixes and resets safely after rotation.
// Pattern: Observable allocation benchmark and capture contract tests.
// Usage: go test -bench BenchmarkCaptureAppend -benchmem ./internal/tools/local/process.
// Related: internal/tools/local/process/capture_changes.go, internal/tools/local/process/output.go.
package process

import (
	"bytes"
	"strings"
	"testing"
)

func TestCaptureAppendAndRotation(t *testing.T) {
	for _, directory := range []string{"", t.TempDir()} {
		var o cappedOutput
		o.directory = directory
		t.Cleanup(func() { _ = o.release() })
		_, _ = o.Write(bytes.Repeat([]byte("x"), 8<<20))
		before := o.changesLocked(0, 0)
		_, _ = o.Write([]byte("next"))
		delta := o.changesLocked(before.Revision, before.Total)
		if delta.Text != "next" || !delta.Append {
			t.Fatal("retained prefix was recopied", len(delta.Text))
		}
		_, _ = o.Write(bytes.Repeat([]byte("y"), 16<<20))
		rotated := o.changesLocked(delta.Revision, delta.Total)
		if rotated.Append || !strings.HasSuffix(rotated.Text, "yyyy") {
			t.Fatal("rotation used an invalid append offset")
		}
	}
}

func BenchmarkCaptureAppend(b *testing.B) {
	var o cappedOutput
	_, _ = o.Write(bytes.Repeat([]byte("x"), 8<<20))
	previous := o.changesLocked(0, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = o.Write([]byte("x"))
		previous = o.changesLocked(previous.Revision, previous.Total)
	}
}

func BenchmarkCaptureFullSnapshot(b *testing.B) {
	var o cappedOutput
	_, _ = o.Write(bytes.Repeat([]byte("x"), 8<<20))
	previous := o.snapshot(0)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = o.Write([]byte("x"))
		previous = o.snapshot(previous.Revision)
	}
}
