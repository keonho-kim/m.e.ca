// Purpose: Incrementally mask response snapshots while withholding incomplete credentials and UTF-8 characters.
// Pattern: Response-owned stream accumulator.
// Usage: Runtime supplies append-only snapshots and creates fresh reply/reasoning masks on retry.
// Related: internal/redaction/matcher.go, internal/agent/run/generate.go.
package redaction

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (v *Registry) NewStreamMask() func(string, bool) string {
	var output strings.Builder
	consumed := 0
	var index *keyIndex
	changed := false
	return func(text string, final bool) string {
		v.mu.RLock()
		next := v.index
		v.mu.RUnlock()
		if index != next || consumed > len(text) {
			output.Reset()
			consumed, index = 0, next
			changed = false
		}
		// Clean snapshots keep sharing the caller's immutable string. A mask,
		// removed control or invalid rune takes ownership of a sanitized copy.
		change := func() {
			if !changed {
				output.WriteString(text[:consumed])
				changed = true
			}
		}
		for consumed < len(text) {
			rest := text[consumed:]
			matched, partial := index.match(rest)
			if partial && !final {
				break
			}
			if matched > 0 {
				change()
				output.WriteString("[redacted]")
				consumed += matched
				continue
			}
			if partial {
				change()
				output.WriteString("[redacted]")
				consumed = len(text)
				break
			}
			if !final && !utf8.FullRuneInString(rest) {
				break
			}
			r, size := utf8.DecodeRuneInString(rest)
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				change()
			} else {
				if r == utf8.RuneError && size == 1 {
					change()
				}
				if changed {
					output.WriteRune(r)
				}
			}
			consumed += size
		}
		if !changed {
			return text[:consumed]
		}
		return output.String()
	}
}
