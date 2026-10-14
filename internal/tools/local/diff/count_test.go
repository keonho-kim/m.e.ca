// Purpose: Verify review size counts actual added and deleted lines independently of preview fallback.
// Pattern: Diff contract tests.
// Usage: go test ./internal/tools/local/diff -run TestCount.
// Related: internal/tools/local/diff/count.go, internal/agent/conclave/vote.go.
package diff

import (
	"strings"
	"testing"
)

func TestCountChanges(t *testing.T) {
	for _, tc := range []struct {
		before, after  string
		added, deleted int
	}{
		{"", "", 0, 0}, {"a\n", "b\n", 1, 1}, {"a", "a\n", 1, 1},
		{strings.Repeat("same\n", 70000) + "a\n", strings.Repeat("same\n", 70000) + "b\n", 1, 1},
		{strings.Repeat("old\n", 40), strings.Repeat("new\n", 40), 40, 40},
		{"one\ntwo\nthree\n", "one\nadded\ntwo\nthree\n", 1, 0},
	} {
		added, deleted, err := Count([]byte(tc.before), []byte(tc.after))
		if err != nil || added != tc.added || deleted != tc.deleted {
			t.Fatalf("count=(%d,%d) want=(%d,%d) err=%v", added, deleted, tc.added, tc.deleted, err)
		}
	}
}

func TestCountMatchesShortestEditsForRepeatedLines(t *testing.T) {
	corpus := []string{""}
	for n := 1; n <= 4; n++ {
		for bits := 0; bits < 1<<n; bits++ {
			text := ""
			for i := 0; i < n; i++ {
				if bits&(1<<i) == 0 {
					text += "a\n"
				} else {
					text += "b\n"
				}
			}
			corpus = append(corpus, text, strings.TrimSuffix(text, "\n"))
		}
	}
	for _, before := range corpus {
		for _, after := range corpus {
			left, right := split(before), split(after)
			row := make([]int, len(right)+1)
			for j := range row {
				row[j] = j
			}
			for i, a := range left {
				diagonal := row[0]
				row[0] = i + 1
				for j, b := range right {
					old := row[j+1]
					if a == b {
						row[j+1] = diagonal
					} else {
						row[j+1] = 1 + min(row[j], old)
					}
					diagonal = old
				}
			}
			added, deleted, err := Count([]byte(before), []byte(after))
			if err != nil || added+deleted != row[len(right)] || added-deleted != len(right)-len(left) {
				t.Fatalf("%q -> %q: %d+%d want %d; %v", before, after, added, deleted, row[len(right)], err)
			}
		}
	}
}
