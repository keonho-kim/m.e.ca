// Purpose: Compare indexed stream privacy with complete-value masking over arbitrary chunk boundaries.
// Pattern: Deterministic fuzz boundary oracle.
// Usage: go test ./internal/config/credentials -fuzz FuzzIndexedStreamMask -fuzztime=5s.
// Related: internal/redaction/matcher.go, internal/redaction/stream.go.
package redaction

import (
	"testing"
	"unicode/utf8"
)

func FuzzIndexedStreamMask(f *testing.F) {
	v := &Registry{}
	for _, key := range []string{"sk-shared", "sk-shared-long", "sk-other", "한글키", "élève", "éclair", "x", "xy"} {
		if err := v.Remember(key); err != nil {
			f.Fatal(err)
		}
	}
	for _, seed := range []string{"ordinary text", "sk-shared-long", "s\x00k-shared", "한\x00글키", "\xc3\x00\xa9clair", "xyx"} {
		f.Add([]byte(seed), uint8(3))
	}
	f.Fuzz(func(t *testing.T, raw []byte, step uint8) {
		if len(raw) > 4096 {
			t.Skip()
		}
		text := string(raw) + "!"
		mask := v.NewStreamMask()
		for at := 1; at < len(text); at += int(step) + 1 {
			mask(text[:at], false)
		}
		got := mask(text, true)
		if !utf8.ValidString(got) || v.Redact(got) != got {
			t.Fatal("masked output exposed a credential or invalid UTF-8")
		}
		// Invalid byte sequences may be masked conservatively when controls
		// separate bytes of a credential; valid text must remain exact.
		if utf8.Valid(raw) && got != v.Redact(text) {
			t.Fatalf("masked stream differs: got=%q want=%q", got, v.Redact(text))
		}
	})
}
