// Purpose: Retain a UTF-8 prefix of authenticated JSON bytes exposed as original text.
// Pattern: Bounded streaming text projection.
// Usage: Canonical JSON payload references share storage between tool messages and events.
// Related: internal/storage/content/prefix_reader.go, internal/storage/content/reference.go.
package content

import "unicode/utf8"

type rawPrefix struct {
	body         []byte
	limit, total int
}

func (p *rawPrefix) write(body []byte) error {
	p.total += len(body)
	if keep := min(len(body), p.limit+utf8.UTFMax-len(p.body)); keep > 0 {
		p.body = append(p.body, body[:keep]...)
	}
	return nil
}

func (p *rawPrefix) finish(limit int) (string, bool, error) {
	end := min(limit, len(p.body))
	for end > 0 && end < len(p.body) && !utf8.RuneStart(p.body[end]) {
		end--
	}
	return string(p.body[:end]), end < p.total, nil
}

func rawTextPrefix(raw []byte, limit int) (string, bool, error) {
	p := rawPrefix{limit: limit}
	_ = p.write(raw)
	return p.finish(limit)
}
