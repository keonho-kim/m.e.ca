// Purpose: Validate a complete JSON string while retaining only enough encoded text for a preview.
// Pattern: Bounded incremental JSON string state machine.
// Usage: Codec.TextPrefix feeds decompressed chunks before publishing any visible text.
// Related: internal/storage/content/prefix_reader.go, internal/storage/content/text_prefix.go.
package content

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type prefixString struct {
	state, digits             byte
	encoded                   []byte
	safe, capacity, remaining int
}

func newPrefixString(size, limit int) prefixString {
	// Six encoded bytes can represent one visible byte (e.g. \u0000).
	// The extra escape leaves enough text to distinguish an exact boundary.
	remaining := min(limit, MaxBytes) + 4
	capacity := int(min(int64(size), 6*int64(remaining)+2))
	return prefixString{capacity: capacity, remaining: remaining, encoded: make([]byte, 0, min(capacity+1, min(limit, MaxBytes)+16))}
}

func (p *prefixString) write(body []byte) error {
	for len(body) > 0 {
		if p.state == 1 {
			end := 0
			for end < len(body) && body[end] >= 0x20 && body[end] != '"' && body[end] != '\\' {
				end++
			}
			if end > 0 {
				if keep := min(end, p.remaining, p.capacity-len(p.encoded)); keep > 0 {
					p.encoded = append(p.encoded, body[:keep]...)
					p.safe = len(p.encoded)
					p.remaining -= keep
				}
				body = body[end:]
				continue
			}
		}
		value := body[0]
		body = body[1:]
		quoted := p.state != 0
		switch p.state {
		case 0: // Leading JSON whitespace.
			if jsonSpace(value) {
				continue
			}
			if value != '"' {
				return faults.Errorf(diagcatalog.ContentStoredTextMustBeAJSONString)
			}
			p.state = 1
		case 1: // Quoted string contents.
			switch {
			case value == '"':
				p.state = 4
			case value == '\\':
				p.state = 2
			case value < 0x20:
				return faults.Errorf(diagcatalog.ContentInvalidStoredTextControlCharacter)
			}
		case 2: // A JSON escape.
			switch value {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				p.state = 1
			case 'u':
				p.state, p.digits = 3, 4
			default:
				return faults.Errorf(diagcatalog.ContentInvalidStoredTextEscape)
			}
		case 3: // Four hexadecimal Unicode escape digits.
			if !(value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F') {
				return faults.Errorf(diagcatalog.ContentInvalidStoredTextUnicodeEscape)
			}
			p.digits--
			if p.digits == 0 {
				p.state = 1
			}
		case 4: // Nothing except whitespace may follow the closing quote.
			if !jsonSpace(value) {
				return faults.Errorf(diagcatalog.ContentInvalidStoredTextSuffix)
			}
			continue
		}
		if p.remaining > 0 && len(p.encoded) < p.capacity {
			p.encoded = append(p.encoded, value)
			if p.state == 1 {
				p.safe = len(p.encoded)
				if quoted {
					p.remaining--
				}
			}
		}
	}
	return nil
}

func (p *prefixString) finish(limit int) (string, bool, error) {
	if p.state != 4 {
		return "", false, faults.Errorf(diagcatalog.ContentIncompleteStoredText)
	}
	// Drop only an unfinished escape at the retention boundary. At least
	// limit+1 decoded bytes fit before that boundary whenever text was clipped.
	raw := append(p.encoded[:p.safe], '"')
	return StringPrefix(raw, limit)
}

func jsonSpace(value byte) bool {
	return value == ' ' || value == '\n' || value == '\r' || value == '\t'
}
