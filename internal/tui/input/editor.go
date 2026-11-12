// Purpose: Own deterministic Unicode input editing and shared immutable text/display snapshots.
// Pattern: Simple Module (deterministic input state).
// Usage: The terminal event loop owns and mutates one editor per input field.
// Related: internal/tui/input/navigation.go, internal/tui/input/layout.go, internal/tui/session/keys.go.
package input

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 16 * 1024 * 1024

type Editor struct {
	runes        []rune
	Cursor       int
	bytes        int
	text         string
	display      string // Immutable prefixed snapshot sharing the body backing string.
	textCapacity int    // Backing snapshot size, including prefixes, before suffix deletions.
	dirty        bool
	revision     uint64
	layout       editorLayout
	column       int
	vertical     bool
	boundaries   []boundaryPoint
	prefix       string
}

func (e *Editor) String() string {
	if e.dirty {
		e.text, e.dirty = string(e.runes), false
		e.textCapacity = len(e.text)
	}
	return e.text
}
func (e *Editor) Revision() uint64 { return e.revision }
func (e *Editor) Len() int         { return len(e.runes) }
func (e *Editor) Before() string {
	if e.Cursor == len(e.runes) {
		return e.String()
	}
	return string(e.runes[:e.Cursor])
}
func (e *Editor) Set(s string) {
	e.runes = []rune(s)
	if !utf8.ValidString(s) {
		s = string(e.runes)
	}
	e.Cursor = len(e.runes)
	e.text, e.bytes, e.dirty = s, len(s), false
	e.textCapacity = len(s)
	e.display = ""
	e.revision++
	e.vertical = false
	e.layout = editorLayout{}
	e.boundaries = nil
}
func (e *Editor) Insert(s string) {
	if !utf8.ValidString(s) || e.bytes+len(s) > MaxBytes {
		return
	}
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
	e.replace(e.Cursor, e.Cursor, s)
}
func (e *Editor) replace(start, end int, s string) {
	if start == end && s == "" {
		return
	}
	e.invalidateBoundaries(start)
	e.invalidateLayout(start)
	r := []rune(s)
	removed := 0
	for _, value := range e.runes[start:end] {
		removed += utf8.RuneLen(value)
	}
	if !e.dirty && end == len(e.runes) {
		prefix := e.prompt()
		if s == "" {
			e.text = e.text[:e.bytes-removed]
			if e.display != "" {
				e.display = e.display[:len(prefix)+len(e.text)]
			}
		} else {
			e.display = prefix + e.text[:e.bytes-removed] + s
			e.text = e.display[len(prefix):]
			e.textCapacity = len(e.display)
		}
		if len(e.text) == 0 || e.textCapacity > 4*max(1024, len(e.text)) {
			e.text = strings.Clone(e.text)
			e.display = ""
			e.textCapacity = len(e.text)
		}
	} else {
		e.text, e.dirty = "", true
		e.display = ""
	}
	e.bytes += len(s) - removed
	old := len(e.runes)
	next := old - (end - start) + len(r)
	if next > old {
		e.runes = append(e.runes, make([]rune, next-old)...)
	}
	copy(e.runes[start+len(r):], e.runes[end:old])
	copy(e.runes[start:], r)
	clear(e.runes[next:])
	e.runes = e.runes[:next]
	if next == 0 {
		e.runes = nil
	} else if cap(e.runes) > 4*max(1024, next) {
		e.runes = slices.Clone(e.runes)
	}
	e.Cursor = start + len(r)
	e.vertical = false
	e.revision++
}
func (e *Editor) Home() {
	e.vertical = false
	for e.Cursor > 0 && e.runes[e.Cursor-1] != '\n' {
		e.Cursor--
	}
}
func (e *Editor) End() {
	e.vertical = false
	for e.Cursor < len(e.runes) && e.runes[e.Cursor] != '\n' {
		e.Cursor++
	}
}
func (e *Editor) DeleteWord() {
	end := e.Cursor
	for e.Cursor > 0 && unicode.IsSpace(e.runes[e.Cursor-1]) {
		e.Cursor--
	}
	for e.Cursor > 0 && !unicode.IsSpace(e.runes[e.Cursor-1]) {
		e.Cursor--
	}
	e.replace(e.Cursor, end, "")
}
