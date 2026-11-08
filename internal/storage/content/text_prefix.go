// Purpose: Decode a bounded visible prefix without materializing or re-encoding a whole message.
// Pattern: Validated JSON string projection.
// Usage: SQLite server previews retain payload integrity checks and preserve incomplete drafts.
// Related: internal/storage/content/text.go, internal/storage/sqlite/timeline.go.
package content

import (
	"bytes"
	"encoding/json"

	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TextPrefix(body, refs []byte, load func(string, int) (string, bool, error), limit int) (string, bool, error) {
	raw, reference, err := textSource(body, refs, false)
	if err == nil && reference != "" {
		return load(reference, limit)
	}
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return "", false, err
	}
	return StringPrefix(raw, limit)
}

// StringPrefix validates one encoded JSON string and returns its bounded visible text.
func StringPrefix(raw []byte, limit int) (string, bool, error) {
	raw = bytes.TrimSpace(raw)
	if limit < 1 || !json.Valid(raw) || len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false, faults.Errorf(diagcatalog.ContentInvalidStoredTextPrefix)
	}
	end := min(limit, len(raw)-2) + 1
	for end > 1 && end < len(raw)-1 && !utf8.RuneStart(raw[end]) {
		end--
	}
	if bytes.IndexByte(raw[1:end], '\\') < 0 && utf8.Valid(raw[1:end]) {
		return string(raw[1:end]), end < len(raw)-1, nil
	}
	var text strings.Builder
	text.Grow(min(limit, len(raw)))
	for at := 1; at < len(raw)-1; {
		r, size := utf8.DecodeRune(raw[at:])
		if raw[at] == '\\' {
			size = 2
			switch raw[at+1] {
			case '"', '\\', '/':
				r = rune(raw[at+1])
			case 'b':
				r = '\b'
			case 'f':
				r = '\f'
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case 'u':
				value, _ := strconv.ParseUint(string(raw[at+2:at+6]), 16, 16)
				r, size = rune(value), 6
				if utf16.IsSurrogate(r) {
					if r >= 0xD800 && r <= 0xDBFF && at+12 <= len(raw)-1 && string(raw[at+6:at+8]) == `\u` {
						low, _ := strconv.ParseUint(string(raw[at+8:at+12]), 16, 16)
						if low >= 0xDC00 && low <= 0xDFFF {
							r, size = utf16.DecodeRune(r, rune(low)), 12
						} else {
							r = utf8.RuneError
						}
					} else {
						r = utf8.RuneError
					}
				}
			}
		}
		if text.Len()+utf8.RuneLen(r) > limit {
			return text.String(), true, nil
		}
		text.WriteRune(r)
		at += size
	}
	return text.String(), false, nil
}
