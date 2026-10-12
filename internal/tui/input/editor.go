// Purpose: Own Unicode input editing and literal Markdown path references.
// Pattern: Simple Module (deterministic input state).
// Usage: The terminal event loop owns and mutates one editor per input field.
// Related: internal/tui/input/editor_test.go, internal/tui/session/keys.go.
package input

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 64 * 1024

type Editor struct {
	Text   []rune
	Cursor int
}

func (e *Editor) String() string { return string(e.Text) }
func (e *Editor) Set(s string)   { e.Text = []rune(s); e.Cursor = len(e.Text) }
func (e *Editor) Insert(s string) {
	if !utf8.ValidString(s) || len(e.String())+len(s) > MaxBytes {
		return
	}
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
	e.Replace(e.Cursor, e.Cursor, s)
}
func (e *Editor) Replace(start, end int, s string) {
	r := []rune(s)
	next := append([]rune(nil), e.Text[:start]...)
	next = append(next, r...)
	next = append(next, e.Text[end:]...)
	e.Text = next
	e.Cursor = start + len(r)
}
func (e *Editor) Move(n int) { e.Cursor = max(0, min(len(e.Text), e.Cursor+n)) }
func (e *Editor) Home() {
	for e.Cursor > 0 && e.Text[e.Cursor-1] != '\n' {
		e.Cursor--
	}
}
func (e *Editor) End() {
	for e.Cursor < len(e.Text) && e.Text[e.Cursor] != '\n' {
		e.Cursor++
	}
}
func (e *Editor) Backspace() {
	if e.Cursor > 0 {
		e.Replace(e.Cursor-1, e.Cursor, "")
	}
}
func (e *Editor) Delete() {
	if e.Cursor < len(e.Text) {
		e.Replace(e.Cursor, e.Cursor+1, "")
	}
}
func (e *Editor) DeleteWord() {
	end := e.Cursor
	for e.Cursor > 0 && unicode.IsSpace(e.Text[e.Cursor-1]) {
		e.Cursor--
	}
	for e.Cursor > 0 && !unicode.IsSpace(e.Text[e.Cursor-1]) {
		e.Cursor--
	}
	e.Replace(e.Cursor, end, "")
}
func (e *Editor) Reference() (token string, start int, ok bool) {
	start = e.Cursor
	for start > 0 && !unicode.IsSpace(e.Text[start-1]) {
		start--
	}
	if start < e.Cursor && e.Text[start] == '@' {
		return string(e.Text[start+1 : e.Cursor]), start, true
	}
	return "", 0, false
}
func Link(label, path string) string {
	label = strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]").Replace(label)
	parts := strings.Split(path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
		parts[i] = strings.NewReplacer("(", "%28", ")", "%29").Replace(parts[i])
	}
	return "[" + label + "](" + strings.Join(parts, "/") + ")"
}
