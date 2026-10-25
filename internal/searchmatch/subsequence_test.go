// Purpose: Verify abbreviation matching keeps order and Unicode characters intact.
// Pattern: Pure contract tests.
// Usage: go test ./internal/searchmatch.
// Related: internal/searchmatch/subsequence.go.
package searchmatch

import "testing"

func TestSubsequence(t *testing.T) {
	for _, tc := range []struct {
		query, value string
		want         bool
	}{
		{"mdl", "Model", true}, {"q38", "Qwen-3.8", true}, {"모설", "모델 설정", true},
		{"Σδ", "Συνδεσηδ", true}, {"aa", "a", false}, {"8qw", "Qwen-3.8", false}, {"", "model", false},
	} {
		if got := Subsequence(tc.query, tc.value); got != tc.want {
			t.Fatalf("%q in %q: %v", tc.query, tc.value, got)
		}
	}
}
