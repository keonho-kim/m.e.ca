// Purpose: Incrementally mask response snapshots while withholding incomplete credentials and UTF-8 characters.
// Pattern: Response-owned stream accumulator.
// Usage: Runtime supplies append-only snapshots and creates fresh reply/reasoning masks on retry.
// Related: internal/config/credentials/redaction.go, internal/agent/run/generate.go.
package credentials

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (v *Vault) NewStreamMask() func(string, bool) string {
	var output strings.Builder
	consumed, keyCount := 0, 0
	return func(text string, final bool) string {
		v.mu.RLock()
		defer v.mu.RUnlock()
		if keyCount != len(v.keys) || consumed > len(text) {
			output.Reset()
			consumed, keyCount = 0, len(v.keys)
		}
		for consumed < len(text) {
			rest := text[consumed:]
			matched, partial := 0, false
			for _, key := range v.keys {
				if strings.HasPrefix(rest, key) {
					matched = max(matched, len(key))
				} else if strings.HasPrefix(key, rest) {
					partial = true
				}
			}
			if partial && !final {
				break
			}
			if matched > 0 {
				output.WriteString("[redacted]")
				consumed += matched
				continue
			}
			if partial {
				output.WriteString("[redacted]")
				consumed = len(text)
				break
			}
			if !final && !utf8.FullRuneInString(rest) {
				break
			}
			r, size := utf8.DecodeRuneInString(rest)
			if !unicode.IsControl(r) || r == '\n' || r == '\t' {
				output.WriteRune(r)
			}
			consumed += size
		}
		return output.String()
	}
}
