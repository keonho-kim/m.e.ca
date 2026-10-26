// Purpose: Verify cached grapheme navigation against complete parsing after arbitrary edits.
// Pattern: Differential editor contract tests.
// Usage: go test ./internal/tui/input -run 'TestBoundary|FuzzBoundary'.
// Related: internal/tui/input/boundaries.go, internal/tui/input/editor.go.
package input

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func referenceBoundary(text string, position int, next bool) int {
	count := 0
	g := uniseg.NewGraphemes(text)
	for g.Next() {
		end := count + utf8.RuneCountInString(g.Str())
		if next && end > position {
			return end
		}
		if !next && end >= position {
			return count
		}
		count = end
	}
	return utf8.RuneCountInString(text)
}

func checkBoundaries(t *testing.T, e *Editor) {
	t.Helper()
	text := e.String()
	// Reverse iteration also exercises seeking to checkpoints out of order.
	for position := e.Len(); position >= 0; position-- {
		for _, next := range []bool{false, true} {
			if got, want := e.boundary(position, next), referenceBoundary(text, position, next); got != want {
				t.Fatalf("boundary at %d next=%t: %d want %d", position, next, got, want)
			}
		}
	}
}

func TestBoundaryEdits(t *testing.T) {
	for _, text := range []string{
		strings.Repeat("x", 1024),
		strings.Repeat("한글 á 👩‍💻 🇰🇷 ", 50),
		strings.Repeat("🇰", 600),
		strings.Repeat("\u0600á", 300),
	} {
		var e Editor
		e.Set(text)
		checkBoundaries(t, &e)
		for _, at := range []int{256, 511, 0, e.Len() / 2} {
			e.MoveTo(at)
			e.Insert("\u0301🇺🇸")
			checkBoundaries(t, &e)
			e.Backspace()
			e.Delete()
			checkBoundaries(t, &e)
		}
		e.Set("")
		if e.boundaries != nil {
			t.Fatal("replacing a draft retained its old index")
		}
	}
}

func TestBoundaryIndexReleasesDeletedTail(t *testing.T) {
	var e Editor
	e.Set(strings.Repeat("x", 64*1024))
	e.boundary(e.Len(), false)
	e.DeleteRange(1024, e.Len())
	if cap(e.boundaries) > 4*len(e.boundaries) {
		t.Fatal("deleted text retained a large navigation index")
	}
	e.DeleteRange(1, e.Len())
	if e.boundaries != nil {
		t.Fatal("short draft retained its old index")
	}
}

func FuzzBoundaryEdits(f *testing.F) {
	f.Add("한글 á 👩‍💻 🇰🇷", "\u0301", uint16(5))
	f.Add(strings.Repeat("🇺", 300), "🇸", uint16(256))
	f.Fuzz(func(t *testing.T, text, insert string, at uint16) {
		if len(text) > 2048 || len(insert) > 512 || !utf8.ValidString(text) || !utf8.ValidString(insert) {
			t.Skip()
		}
		var e Editor
		e.Set(text)
		e.boundary(e.Len(), false)
		e.MoveTo(int(at) % (e.Len() + 1))
		e.Insert(insert)
		checkBoundaries(t, &e)
		e.Backspace()
		e.Delete()
		checkBoundaries(t, &e)
	})
}
