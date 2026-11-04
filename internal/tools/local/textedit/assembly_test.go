// Purpose: Verify large disjoint edit assembly preserves original positions and input ownership.
// Pattern: Differential exact-text assembly regression test and benchmark.
// Usage: go test ./internal/tools/local/textedit -run TestBatch -bench BenchmarkBatchEdits.
// Related: internal/tools/local/textedit/ranges.go, internal/tools/local/textedit/edit.go.
package textedit

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func editFixture(size, count int) ([]byte, []Edit) {
	var source strings.Builder
	edits := make([]Edit, count)
	for i := range count {
		source.WriteString(strings.Repeat("regular source line\n", size/count/20))
		old := fmt.Sprintf("unique source marker %08d\n", i)
		source.WriteString(old)
		edits[i] = Edit{OldText: old, NewText: fmt.Sprintf("updated marker %08d\n", i)}
	}
	return []byte(source.String()), edits
}

func TestBatchAssemblyPreservesUnsortedEditsAndOriginalPositions(t *testing.T) {
	before, edits := editFixture(1<<20, 64)
	original := bytes.Clone(before)
	for left, right := 0, len(edits)-1; left < right; left, right = left+1, right-1 {
		edits[left], edits[right] = edits[right], edits[left]
	}
	want := string(before)
	for _, edit := range edits {
		want = strings.ReplaceAll(want, edit.OldText, edit.NewText)
	}
	after, positions, err := ReplaceMany("source.txt", before, edits, "preserve")
	if err != nil || string(after) != want || !bytes.Equal(before, original) {
		t.Fatal("assembly changed source ownership or output", err)
	}
	for i, edit := range edits {
		old := 1 + strings.Count(string(before[:bytes.Index(before, []byte(edit.OldText))]), "\n")
		next := 1 + strings.Count(want[:strings.Index(want, edit.NewText)], "\n")
		if positions[i].OldStartLine != old || positions[i].NewStartLine != next {
			t.Fatalf("incorrect positions for edit %d: %+v", i, positions[i])
		}
	}
}

func BenchmarkBatchEdits(b *testing.B) {
	for _, count := range []int{1, 16, 64} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			before, edits := editFixture(8<<20, count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out, positions, err := ReplaceMany("source.txt", before, edits, "preserve")
				if err != nil || len(out) == 0 || len(positions) != count {
					b.Fatal(err)
				}
			}
		})
	}
}
