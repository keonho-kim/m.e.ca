// Purpose: Reuse authenticated visible prefixes until their last reference in a history page.
// Pattern: Page-owned projection cache sharing the payload retention budget.
// Usage: TimelineSummaries avoids restoring or charging duplicate message content repeatedly.
// Related: internal/storage/sqlite/page_payloads.go, internal/storage/sqlite/text_prefix.go.
package sqlite

type textPrefix struct {
	text    string
	clipped bool
}

func (p *pagePayloads) prefix(id string, limit int) (string, bool, error) {
	key := payloadKey{p.store, id}
	value, cached := p.prefixes[key]
	if !cached {
		text, clipped, err := p.store.readTextPrefix(p.ctx, id, p.budget, limit)
		if err != nil {
			return "", false, err
		}
		value = textPrefix{text: text, clipped: clipped}
	}
	if p.last[key] <= p.index {
		if cached {
			p.bytes -= len(value.text)
			delete(p.prefixes, key)
		}
	} else if !cached && len(value.text) <= pagePayloadBytes-p.bytes {
		if p.prefixes == nil {
			p.prefixes = make(map[payloadKey]textPrefix)
		}
		p.prefixes[key] = value
		p.bytes += len(value.text)
	}
	return value.text, value.clipped, nil
}
