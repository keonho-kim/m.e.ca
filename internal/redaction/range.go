// Purpose: Mask a selected text interval using surrounding bytes from the same credential snapshot.
// Pattern: Pure redaction projection over an immutable matcher.
// Usage: Skill reference pages include bounded lookaround before calling RedactRange.
// Related: internal/redaction/matcher.go, internal/skills/resources.go.
package redaction

import (
	"strings"
	"unicode"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const MaxCredentialBytes = 2048

func (v *Registry) RedactRange(text string, start, end int) (string, error) {
	if start < 0 || end < start || end > len(text) {
		return "", faults.Errorf(diagcatalog.RedactionInvalidRedactionInterval)
	}
	v.mu.RLock()
	index := v.index
	v.mu.RUnlock()
	var output strings.Builder
	for offset := 0; offset < len(text) && offset < end; {
		matched, _ := index.match(text[offset:])
		if matched > 0 {
			if offset+matched > start {
				output.WriteString("[redacted]")
			}
			offset += matched
			continue
		}
		r, n := utf8.DecodeRuneInString(text[offset:])
		if offset >= start && (!unicode.IsControl(r) || r == '\n' || r == '\t') {
			output.WriteRune(r)
		}
		offset += n
	}
	return output.String(), nil
}
