// Purpose: Verify bounded command capture retains final diagnostics as well as the log beginning.
// Pattern: Output contract tests.
// Usage: go test ./internal/tools/local/process.
// Related: internal/tools/local/process/output.go, internal/tools/local/command.go.
package process

import (
	"context"
	"os"
	"runtime"
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

func BenchmarkCommandCapture64MiB(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		b.Skip("requires the supported host shell")
	}
	root := b.TempDir()
	b.ReportAllocs()
	for b.Loop() {
		out, err := Run(context.Background(), root, "head -c 67108864 /dev/zero", os.Environ())
		if err != nil || out.ExitCode != 0 {
			b.Fatal(err, out.ExitCode)
		}
	}
}

func TestOutputRingUnevenChunks(t *testing.T) {
	source := strings.Repeat("0123456789abcdefghijklmnopqrstuvwxyz", captureHalf*3/36+10)
	var out cappedOutput
	for at := 0; at < len(source); at += 32003 {
		if _, err := out.Write([]byte(source[at:min(len(source), at+32003)])); err != nil {
			t.Fatal(err)
		}
	}
	want := source[:captureHalf] + "\n[... output omitted ...]\n" + source[len(source)-captureHalf:]
	if got := out.String(); got != want {
		t.Fatal("ring changed captured head or tail")
	}
}

var capturedBenchmarkOutput string

func BenchmarkCappedOutput64MiB(b *testing.B) {
	chunk := []byte(strings.Repeat("x", 32*1024))
	b.SetBytes(64 * 1024 * 1024)
	b.ReportAllocs()
	for b.Loop() {
		var out cappedOutput
		for range 2048 {
			out.Write(chunk)
		}
		capturedBenchmarkOutput = out.String()
	}
}
