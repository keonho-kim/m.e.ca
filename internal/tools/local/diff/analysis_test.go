// Purpose: Verify shared diff analysis preserves exact counts, context and bounded fallbacks.
// Pattern: Change metadata contract tests.
// Usage: go test ./internal/tools/local/diff -run TestAnalysis.
// Related: internal/tools/local/diff/diff.go, internal/tools/local/diff/count.go.
package diff

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAnalysisMatchesExactCount(t *testing.T) {
	for _, pair := range [][2]string{
		{"", ""}, {"a", "a\n"}, {"a\r\n", "a\n"},
		{"a\nb\na\nc\n", "b\na\na\nd\n"},
		{"", strings.Repeat("new\n", 1000)},
		{strings.Repeat("old\n", 1000), ""},
		{strings.Repeat("old\n", 300) + "same\n", strings.Repeat("new\n", 300) + "same\n"},
		{strings.Repeat("same\n", 70000) + "a\n", strings.Repeat("same\n", 70000) + "b\n"},
	} {
		text, added, deleted, err := Analyze("file", []byte(pair[0]), []byte(pair[1]), true, true)
		wantAdded, wantDeleted, wantErr := Count([]byte(pair[0]), []byte(pair[1]))
		if err != nil || wantErr != nil || text == "" || added != wantAdded || deleted != wantDeleted {
			t.Fatalf("analysis %d/%d/%v want %d/%d/%v", added, deleted, err, wantAdded, wantDeleted, wantErr)
		}
	}
}

func TestAnalysisOmitsDistantUnchangedLines(t *testing.T) {
	prefix, suffix := strings.Repeat("before\n", 70000), strings.Repeat("after\n", 70000)
	preview, added, deleted, err := Analyze("file", []byte(prefix+"old\n"+suffix), []byte(prefix+"new\n"+suffix), true, true)
	if err != nil || added != 1 || deleted != 1 || !strings.Contains(preview, "@@ -69998,7 +69998,7 @@") || strings.Count(preview, "before\n") != 3 || strings.Count(preview, "after\n") != 3 {
		t.Fatalf("wrong trimmed context or line numbers: %q, %v", preview, err)
	}
}

func FuzzAnalysisCounts(f *testing.F) {
	f.Add("a\nb\na\nc\n", "b\na\na\nd\n")
	f.Add(strings.Repeat("before\n", 700)+"old\n", strings.Repeat("before\n", 700)+"new\n")
	f.Add("x\r\n", "x\n")
	f.Fuzz(func(t *testing.T, before, after string) {
		if len(before)+len(after) > 16*1024 || !utf8.ValidString(before) || !utf8.ValidString(after) {
			t.Skip()
		}
		_, added, deleted, err := Analyze("file", []byte(before), []byte(after), true, true)
		wantAdded, wantDeleted, wantErr := Count([]byte(before), []byte(after))
		if (err == nil) != (wantErr == nil) || err == nil && (added != wantAdded || deleted != wantDeleted) {
			t.Fatalf("shared analysis: %d/%d/%v want %d/%d/%v", added, deleted, err, wantAdded, wantDeleted, wantErr)
		}
	})
}
