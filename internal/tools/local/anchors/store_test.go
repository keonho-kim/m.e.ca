// Purpose: Verify short anchors preserve exact range contents and report resulting line positions.
// Pattern: Snapshot and edit contract tests.
// Usage: go test ./internal/tools/local/anchors.
// Related: internal/tools/local/anchors/store.go, internal/tools/local/anchors/edit.go.
package anchors

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestRangeEditChecksMiddleAndPreservesOutsideChanges(t *testing.T) {
	var store Store
	rendered, err := store.Record("/a", 2, sha256.Sum256([]byte("outside\n")), "first\nmiddle\nlast\n")
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")
	start, _, _ := strings.Cut(rows[0], " | ")
	end, _, _ := strings.Cut(rows[2], " | ")
	if len(strings.Split(start, "#")[1]) != 8 {
		t.Fatal(start)
	}
	if _, _, err := store.Replace("/a", []byte("outside\nfirst\nCHANGED\nlast\n"), start, end, "new\n", "preserve"); err == nil {
		t.Fatal("middle change accepted")
	}
	after, span, err := store.Replace("/a", []byte("outside\nfirst\nmiddle\nlast\ntail changed\n"), start, end, "new\nsecond\n", "preserve")
	if err != nil || string(after) != "outside\nnew\nsecond\ntail changed\n" || span.OldStartLine != 2 || span.OldEndLine != 4 || span.NewStartLine != 2 || span.NewEndLine != 3 {
		t.Fatalf("%q %+v %v", after, span, err)
	}
}
func TestAnchorsCannotMixReadsOrFiles(t *testing.T) {
	var store Store
	first, err := store.Record("/a", 1, sha256.Sum256(nil), "one\ntwo\n")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Record("/a", 1, sha256.Sum256(nil), "one\ntwo\nthree\n")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Split(first, " | ")[0]
	end := strings.Split(strings.Split(second, "\n")[1], " | ")[0]
	if _, _, err := store.Replace("/a", []byte("one\ntwo\n"), start, end, "new\n", "preserve"); err == nil {
		t.Fatal("mixed snapshots accepted")
	}
	if _, _, err := store.Replace("/b", []byte("one\ntwo\n"), start, start, "new\n", "preserve"); err == nil {
		t.Fatal("cross-file anchor accepted")
	}
	store.Clear()
	if _, _, err := store.Replace("/a", []byte("one\ntwo\n"), start, start, "new\n", "preserve"); err == nil {
		t.Fatal("expired anchor accepted")
	}
}

func TestDeleteCRLFAndMovedLineRanges(t *testing.T) {
	for _, tc := range []struct {
		name, before, replacement, after string
		removed                          bool
	}{
		{"CRLF", "one\r\ntwo\r\n", "new\n", "new\r\ntwo\r\n", false},
		{"delete", "one\ntwo\n", "", "two\n", true},
		{"EOF", "one", "new", "new", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var store Store
			read, err := store.Record("/a", 1, sha256.Sum256(nil), tc.before)
			if err != nil {
				t.Fatal(err)
			}
			anchor := strings.Split(read, " | ")[0]
			after, span, err := store.Replace("/a", []byte(tc.before), anchor, anchor, tc.replacement, "preserve")
			if err != nil || string(after) != tc.after || span.Removed != tc.removed {
				t.Fatalf("%q %+v %v", after, span, err)
			}
			if _, _, err := store.Replace("/a", []byte("inserted\n"+tc.before), anchor, anchor, tc.replacement, "preserve"); err == nil {
				t.Fatal("moved lines were guessed")
			}
		})
	}
}
func TestEvictionAndAmbiguousAnchorsRequireRead(t *testing.T) {
	var store Store
	first, err := store.Record("/a", 1, sha256.Sum256(nil), "one\n")
	if err != nil {
		t.Fatal(err)
	}
	anchor := strings.Split(first, " | ")[0]
	// A short-label collision must not select either matching stored source.
	store.reads = append(store.reads, store.reads[0])
	if _, _, err := store.Replace("/a", []byte("one\n"), anchor, anchor, "new\n", "preserve"); err == nil {
		t.Fatal("ambiguous short anchor accepted")
	}
	store.Clear()
	first, err = store.Record("/a", 1, sha256.Sum256(nil), "one\n")
	if err != nil {
		t.Fatal(err)
	}
	anchor = strings.Split(first, " | ")[0]
	for i := 0; i < maxReads; i++ {
		if _, err := store.Record(fmt.Sprintf("/other/%d", i), 1, sha256.Sum256(nil), "one\n"); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := store.Replace("/a", []byte("one\n"), anchor, anchor, "new\n", "preserve"); err == nil {
		t.Fatal("evicted anchor accepted")
	}
}

func TestIdenticalReadsKeepStableEvidence(t *testing.T) {
	var store Store
	a, err := store.Record("/a", 1, sha256.Sum256(nil), "same\n")
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Record("/a", 1, sha256.Sum256(nil), "same\n")
	if err != nil || a != b {
		t.Fatalf("identical reads fabricated progress: %q %q %v", a, b, err)
	}
	anchor := strings.Split(a, " | ")[0]
	if _, _, err := store.Replace("/a", []byte("same\n"), anchor, anchor, "new\n", "preserve"); err != nil {
		t.Fatal(err)
	}
}

func TestPreserveCompletesLineBoundary(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		before := "name: demo" + newline + "status: pending" + newline + "count: 3" + newline
		var store Store
		rendered, err := store.Record("/a", 1, sha256.Sum256(nil), before)
		if err != nil {
			t.Fatal(err)
		}
		anchor := strings.Split(strings.Split(rendered, "\n")[1], " | ")[0]
		after, _, err := store.Replace("/a", []byte(before), anchor, anchor, "status: done", "preserve")
		if err != nil || string(after) != "name: demo"+newline+"status: done"+newline+"count: 3"+newline {
			t.Fatalf("%q %v", after, err)
		}
		literal, _, err := store.Replace("/a", []byte(before), anchor, anchor, "status: done", "literal")
		if err != nil || string(literal) != "name: demo"+newline+"status: donecount: 3"+newline {
			t.Fatalf("explicit joining failed: %q %v", literal, err)
		}
	}
}
