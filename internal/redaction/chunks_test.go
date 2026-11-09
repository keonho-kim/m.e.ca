// Purpose: Keep chunked masking equivalent to whole-value masking across secret and rune boundaries.
// Pattern: Streaming redaction contract tests.
// Usage: go test ./internal/redaction -run TestChunkMask.
// Related: internal/redaction/chunks.go, internal/redaction/redaction.go.
package redaction

import (
	"strings"
	"testing"
)

func TestChunkMaskMatchesCompleteValue(t *testing.T) {
	var registry Registry
	for _, key := range []string{"secret-key", "secret-key-long"} {
		if err := registry.Remember(key); err != nil {
			t.Fatal(err)
		}
	}
	for _, text := range []string{"before secret-key-long 한국어 after secret-key tail", "secret-", "se\x00cret-key", "한글🙂\x1bmore", strings.Repeat("x", 8191) + "secret-key"} {
		for _, size := range []int{1, 2, 7, 8192} {
			mask := registry.NewChunkMask()
			var output strings.Builder
			for start := 0; start < len(text); start += size {
				output.WriteString(mask(text[start:min(len(text), start+size)], false))
			}
			output.WriteString(mask("", true))
			if got, want := output.String(), registry.Redact(text); got != want {
				t.Fatalf("chunk %d: %q, want %q", size, got, want)
			}
		}
	}
}
