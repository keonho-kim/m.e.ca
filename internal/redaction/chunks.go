// Purpose: Mask independent text chunks while retaining only unfinished credentials and UTF-8 bytes.
// Pattern: Bounded stream transformation over an immutable credential snapshot.
// Usage: Detail JSON writers create one mask per string and flush it at the closing quote.
// Related: internal/redaction/matcher.go, internal/serverapi/detail/write_string.go.
package redaction

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (v *Registry) NewChunkMask() func(string, bool) string {
	v.mu.RLock()
	index := v.index
	v.mu.RUnlock()
	pending, tail := "", ""
	return func(chunk string, final bool) string {
		raw := chunk
		if tail != "" {
			raw = tail + raw
			tail = ""
		}
		end := len(raw)
		if !final {
			start := max(0, end-utf8.UTFMax)
			for start > 0 && !utf8.RuneStart(raw[start]) {
				start--
			}
			for start < end {
				if !utf8.FullRuneInString(raw[start:]) {
					end = start
					break
				}
				_, n := utf8.DecodeRuneInString(raw[start:])
				start += n
			}
			tail = strings.Clone(raw[end:])
		}
		clean := cleanChunk(raw[:end])
		text := clean
		if pending != "" {
			text = pending + clean
			pending = ""
		}
		var output strings.Builder
		changed := false
		offset := 0
		for offset < len(text) {
			matched, partial := index.match(text[offset:])
			if partial && !final {
				break
			}
			if matched > 0 {
				if !changed {
					output.WriteString(text[:offset])
					changed = true
				}
				output.WriteString("[redacted]")
				offset += matched
				continue
			}
			_, n := utf8.DecodeRuneInString(text[offset:])
			if changed {
				output.WriteString(text[offset : offset+n])
			}
			offset += n
		}
		pending = strings.Clone(text[offset:])
		if !changed {
			return text[:offset]
		}
		return output.String()
	}
}

func cleanChunk(text string) string {
	for offset, r := range text {
		if unicode.IsControl(r) && r != '\n' && r != '\t' || r == utf8.RuneError {
			return text[:offset] + strings.Map(func(r rune) rune {
				if unicode.IsControl(r) && r != '\n' && r != '\t' {
					return -1
				}
				return r
			}, text[offset:])
		}
	}
	return text
}
