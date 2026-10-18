// Purpose: Reject ambiguous source and preserve byte formats without a read cache.
// Pattern: Exact-text contract regression tests.
// Usage: go test ./internal/tools/local/textedit.
// Related: internal/tools/local/textedit/edit.go, internal/tools/local/textedit/ranges.go.
package textedit

import (
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestAmbiguousAndMissingSourceReturnSpecificFailures(t *testing.T) {
	for _, tc := range []struct{ source, old, code string }{
		{"aaaa", "aa", "ambiguous_text"},
		{"current", "previous", "text_not_found"},
	} {
		after, _, err := ReplaceMany("/source", []byte(tc.source), []Edit{{tc.old, "new"}}, "preserve")
		if err == nil || toolapi.FailureOf(err).Code != tc.code || after != nil {
			t.Fatalf("%+v: result=%q err=%v", tc, after, err)
		}
	}
}

func TestCRLFAndFinalNewlineRemainPreserved(t *testing.T) {
	after, _, err := ReplaceMany("/source", []byte("a\r\nb\r\n"), []Edit{{"b\r\n", "B\nC"}}, "preserve")
	if err != nil || string(after) != "a\r\nB\r\nC\r\n" {
		t.Fatalf("result=%q err=%v", after, err)
	}
	after, _, err = ReplaceMany("/source", []byte("abc"), []Edit{{"b", ""}}, "preserve")
	if err != nil || string(after) != "ac" {
		t.Fatalf("result=%q err=%v", after, err)
	}
}
