// Purpose: Verify atomic multi-range edits preserve original coordinates and final positions.
// Pattern: Snapshot contract tests.
// Usage: go test ./internal/tools/local/textedit.
// Related: internal/tools/local/textedit/ranges.go.
package textedit

import (
	"testing"
)

func TestDisjointEditsUseOneOriginalAndReportShiftedLines(t *testing.T) {
	before := "a\nb\nc\nd\ne\n"
	edits := []Edit{{"d\n", "D\n"}, {"b\n", "B1\nB2\n"}}
	after, spans, err := ReplaceMany("/a", []byte(before), edits, "preserve")
	if err != nil || string(after) != "a\nB1\nB2\nc\nD\ne\n" || spans[0].NewStartLine != 5 || spans[1].NewEndLine != 3 {
		t.Fatalf("%q %+v %v", after, spans, err)
	}
	edits[0] = Edit{"b\nc\n", "x\n"}
	if after, _, err := ReplaceMany("/a", []byte(before), edits, "preserve"); err == nil || after != nil {
		t.Fatal("overlap returned a partial change")
	}
	edits[0] = Edit{"d\n", "D\n"}
	if after, _, err := ReplaceMany("/a", []byte("a\nb\nc\nDIFFERENT\ne\n"), edits, "preserve"); err == nil || after != nil {
		t.Fatal("stale range returned a partial change")
	}
}

func TestLiteralJoiningKeepsFinalLinePositions(t *testing.T) {
	before := "a\nb\nc\nd\n"
	after, spans, err := ReplaceMany("/a", []byte(before), []Edit{{"a\n", "A"}, {"c\n", "C"}}, "literal")
	if err != nil || string(after) != "Ab\nCd\n" || spans[0].NewEndLine != 1 || spans[1].NewStartLine != 2 {
		t.Fatalf("%q %+v %v", after, spans, err)
	}
}
