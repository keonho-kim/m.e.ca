// Purpose: Verify private spill captures preserve bounded head/tail output and clean up.
// Pattern: Resource lifecycle and differential tests.
// Usage: go test ./internal/tools/local/process.
// Related: internal/tools/local/process/spill.go, internal/tools/local/process/capture_file.go.
package process

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSpillPreservesRingAndRemovesEncryptedFile(t *testing.T) {
	directory := t.TempDir()
	output := cappedOutput{directory: directory}
	t.Cleanup(func() { _ = output.release() })
	source := strings.Repeat("private-canary-0123456789\n", 3*captureHalf/25+1)
	for at := 0; at < len(source); at += 32003 {
		if _, err := output.Write([]byte(source[at:min(len(source), at+32003)])); err != nil {
			t.Fatal(err)
		}
	}
	want := source[:captureHalf] + "\n[... output omitted ...]\n" + source[len(source)-captureHalf:]
	if got := output.String(); got != want {
		t.Fatal("spill changed captured head or tail")
	}
	if !output.isTruncated() || output.head.Len() != 0 || len(output.tail) != 0 {
		t.Fatal("spill retained the memory ring")
	}
	path := output.spill.file.Name()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private file: %v, %v", info, err)
	}
	frame := captureBlock + output.spill.file.aead.NonceSize() + output.spill.file.aead.Overhead()
	if info.Size() > int64(len(output.spill.file.known)*frame) {
		t.Fatalf("unbounded spill: %d", info.Size())
	}
	ciphertext, err := os.ReadFile(path)
	if err != nil || bytes.Contains(ciphertext, []byte("private-canary")) {
		t.Fatalf("raw output reached spill: %v", err)
	}
	if err := output.release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("spill survived release: %v", err)
	}
}

func TestCaptureFileRejectsTampering(t *testing.T) {
	file, err := newCaptureFile(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err = file.WriteAt([]byte("secret"), 0); err != nil {
		t.Fatal(err)
	}
	// Switching blocks flushes the first authenticated frame.
	if _, err = file.WriteAt([]byte("tail"), captureBlock); err != nil {
		t.Fatal(err)
	}
	if _, err = file.file.WriteAt([]byte{42}, int64(file.aead.NonceSize()+10)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err = file.ReadAt(make([]byte, 6), 0); err == nil {
			t.Fatal("corrupt frame was accepted")
		}
	}
}

func TestSpillIOFailureRemainsExplicit(t *testing.T) {
	output := cappedOutput{directory: t.TempDir()}
	t.Cleanup(func() { _ = output.release() })
	if _, err := output.Write(bytes.Repeat([]byte("a"), memoryCaptureBytes+1)); err != nil {
		t.Fatal(err)
	}
	if err := output.spill.file.file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write(bytes.Repeat([]byte("b"), captureBlock)); err == nil || output.captureError() == nil {
		t.Fatal("spill write failure was hidden")
	}
}

func BenchmarkSpillOutput64MiB(b *testing.B) {
	directory := b.TempDir()
	chunk := []byte(strings.Repeat("x", 32*1024))
	b.SetBytes(64 * 1024 * 1024)
	b.ReportAllocs()
	for b.Loop() {
		output := cappedOutput{directory: directory}
		for range 2048 {
			if _, err := output.Write(chunk); err != nil {
				b.Fatal(err)
			}
		}
		capturedBenchmarkOutput = output.String()
		if err := output.release(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSmallCommandCapture(b *testing.B) {
	directory := b.TempDir()
	chunk := []byte(strings.Repeat("x", 4096))
	for _, managed := range []bool{false, true} {
		name := "memory"
		if managed {
			name = "owned"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var output cappedOutput
				if managed {
					output.directory = directory
				}
				for range 4 {
					if _, err := output.Write(chunk); err != nil {
						b.Fatal(err)
					}
				}
				capturedBenchmarkOutput = output.String()
				if err := output.release(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
