// Purpose: Preserve JSON text semantics while allocating only the visible prefix.
// Pattern: Table, fuzz and allocation boundary tests.
// Usage: go test ./internal/storage/content -run TestTextPrefix -bench BenchmarkTextPrefix.
// Related: internal/storage/content/text_prefix.go, internal/storage/sqlite/timeline.go.
package content

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTextPrefixEscapesUnicodeAndIncomplete(t *testing.T) {
	if text, clipped, err := StringPrefix([]byte(`"complete"`), int(^uint(0)>>1)); err != nil || clipped || text != "complete" {
		t.Fatal("large positive preview limit overflowed", text, clipped, err)
	}
	for _, raw := range []string{`"hello"`, `"a\n\/b"`, `"\ud83d\ude00한글"`, `"\ud800x"`, `""`} {
		var decoded string
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatal(err)
		}
		for limit := 1; limit <= len(decoded)+1; limit++ {
			got, clipped, err := StringPrefix([]byte(raw), limit)
			want := decoded[:min(limit, len(decoded))]
			for !utf8.ValidString(want) {
				want = want[:len(want)-1]
			}
			if err != nil || got != want || clipped != (want != decoded) {
				t.Fatal(raw, limit, got, want, clipped, err)
			}
		}
	}
	text, clipped, err := TextPrefix([]byte(`{"content":"unfinished draft","incomplete":true}`), []byte(`{}`), nil, 5)
	if err != nil || text != "unfin" || !clipped {
		t.Fatal(text, clipped, err)
	}
}

func FuzzTextPrefix(f *testing.F) {
	for _, text := range []string{"plain", "한글😀", "\\\"\n", "\x00"} {
		f.Add(text, uint8(7))
	}
	f.Fuzz(func(t *testing.T, text string, length uint8) {
		raw, _ := json.Marshal(text)
		var decoded string
		_ = json.Unmarshal(raw, &decoded)
		limit := int(length) + 1
		got, clipped, err := StringPrefix(raw, limit)
		want := decoded[:min(limit, len(decoded))]
		for !utf8.ValidString(want) {
			want = want[:len(want)-1]
		}
		if err != nil || got != want || clipped != (want != decoded) {
			t.Fatal(got, want, clipped, err)
		}
	})
}

func BenchmarkTextPrefix8MiB(b *testing.B) {
	raw, _ := json.Marshal(strings.Repeat("x", 8<<20))
	b.ReportAllocs()
	for b.Loop() {
		text, clipped, err := StringPrefix(raw, 16<<10)
		if err != nil || len(text) != 16<<10 || !clipped {
			b.Fatal(err)
		}
	}
}
