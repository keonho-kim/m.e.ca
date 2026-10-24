// Purpose: Verify incremental credential masking across chunk, Unicode and credential-update boundaries.
// Pattern: Stateful boundary tests and benchmark.
// Usage: go test ./internal/config/credentials -run TestStreamMask.
// Related: internal/config/credentials/redaction.go, internal/config/credentials/stream.go.
package credentials

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStreamMaskChunkBoundaries(t *testing.T) {
	v := &Vault{}
	v.Remember("provider-secret")
	text := "한글 provider-secret text\x1b end"
	for size := 1; size < 20; size++ {
		mask := v.NewStreamMask()
		var visible string
		for at := 0; at < len(text); at += size {
			visible = mask(text[:min(len(text), at+size)], false)
			if !utf8.ValidString(visible) || strings.Contains(visible, "provider-secret") || strings.ContainsRune(visible, '\x1b') {
				t.Fatalf("size=%d: unsafe snapshot %q", size, visible)
			}
		}
		if got := mask(text, true); got != v.Redact(text) {
			t.Fatalf("size=%d: %q", size, got)
		}
	}
	mask := v.NewStreamMask()
	if got := mask("start provider-sec", true); strings.Contains(got, "provider-sec") {
		t.Fatal("unfinished key prefix leaked")
	}
}

func TestStreamMaskIndependentResponsesAndNewKey(t *testing.T) {
	v := &Vault{}
	first, second := v.NewStreamMask(), v.NewStreamMask()
	first("private-value", false)
	v.Remember("private-value")
	if got := first("private-value!", true); got != "[redacted]!" {
		t.Fatal(got)
	}
	if got := second("another reply", true); got != "another reply" {
		t.Fatal("responses shared state", got)
	}
}

func BenchmarkStreamMask(b *testing.B) {
	v := &Vault{}
	v.Remember("provider-secret")
	text := strings.Repeat("token fragment. ", 256*1024/16)
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	for b.Loop() {
		mask := v.NewStreamMask()
		for at := 0; at < len(text); at += 64 {
			mask(text[:at+64], false)
		}
		mask(text, true)
	}
}
