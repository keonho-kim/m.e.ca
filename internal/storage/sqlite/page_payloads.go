// Purpose: Reuse repeated immutable payloads only until their last reference in a history page.
// Pattern: Page-owned bounded read-through cache.
// Usage: Export and Timeline index references after closing rows, then hydrate selected fields.
// Related: internal/storage/sqlite/payload.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"
	"encoding/json"

	"github.com/keonho-kim/meca/internal/storage/content"
)

// This bounds temporary reuse, not the size of a payload accepted by the store.
const pagePayloadBytes = 2 * 1024 * 1024

type pagePayloads struct {
	store  *Store
	ctx    context.Context
	last   map[string]int
	bodies map[string]json.RawMessage
	index  int
	bytes  int
}

func (p *pagePayloads) include(index int, refsBody []byte) error {
	var err error
	p.last, err = indexPayloadRefs(p.last, index, refsBody)
	return err
}

func (p *pagePayloads) unpack(index int, record storedRecord, fields map[string]bool) ([]byte, error) {
	return p.unpackWith(index, record, fields, func(id string) (json.RawMessage, error) {
		return p.store.payload(p.ctx, id)
	})
}

func (p *pagePayloads) unpackWith(index int, record storedRecord, fields map[string]bool, read func(string) (json.RawMessage, error)) ([]byte, error) {
	p.index = index
	return content.Restore(record.body, record.refs, fields, func(id string) (json.RawMessage, error) {
		return p.load(id, read)
	})
}

func (p *pagePayloads) load(id string, read func(string) (json.RawMessage, error)) (json.RawMessage, error) {
	if err := p.ctx.Err(); err != nil {
		return nil, err
	}
	cached := p.bodies[id]
	body := cached
	if body == nil {
		var err error
		body, err = read(id)
		if err != nil {
			return nil, err
		}
	}
	if p.last[id] <= p.index {
		p.bytes -= len(cached)
		delete(p.bodies, id)
	} else if cached == nil && len(body) <= pagePayloadBytes-p.bytes {
		if p.bodies == nil {
			p.bodies = make(map[string]json.RawMessage)
		}
		p.bodies[id] = body
		p.bytes += len(body)
	}
	return body, nil
}
