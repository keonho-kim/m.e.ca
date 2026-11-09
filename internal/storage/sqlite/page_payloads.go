// Purpose: Reuse immutable payloads and visible prefixes until their last reference in a history page.
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

type payloadKey struct {
	session, id string
}

type pagePayloads struct {
	store    *Store
	ctx      context.Context
	last     map[payloadKey]int
	bodies   map[payloadKey]json.RawMessage
	prefixes map[prefixKey]textPrefix
	index    int
	bytes    int
	budget   *readBudget
}

func (p *pagePayloads) include(index int, store *Store, refsBody []byte) error {
	var err error
	p.last, err = indexPayloadRefs(p.last, store, index, refsBody)
	return err
}

func (p *pagePayloads) unpack(index int, record storedRecord, fields map[string]bool) ([]byte, error) {
	return p.unpackWith(index, payloadSession(p.store), record, fields, func(id string) (json.RawMessage, error) {
		if p.budget != nil {
			return p.store.readPayload(p.ctx, id, p.budget)
		}
		return p.store.payload(p.ctx, id)
	})
}

func (p *pagePayloads) unpackWith(index int, session string, record storedRecord, fields map[string]bool, read func(string) (json.RawMessage, error)) ([]byte, error) {
	p.index = index
	return content.Restore(record.body, record.refs, fields, func(id string) (json.RawMessage, error) {
		return p.load(session, id, read)
	})
}

func (p *pagePayloads) load(session, id string, read func(string) (json.RawMessage, error)) (json.RawMessage, error) {
	if err := p.ctx.Err(); err != nil {
		return nil, err
	}
	key := payloadKey{session, id}
	cached := p.bodies[key]
	body := cached
	if body == nil {
		var err error
		body, err = read(id)
		if err != nil {
			return nil, err
		}
	}
	if p.last[key] <= p.index {
		p.release(key)
	} else if cached == nil && len(body) <= pagePayloadBytes-p.bytes {
		if p.bodies == nil {
			p.bodies = make(map[payloadKey]json.RawMessage)
		}
		p.bodies[key] = body
		p.bytes += len(body)
	}
	return body, nil
}

func payloadSession(store *Store) string {
	if store == nil {
		return ""
	}
	return store.session.ID
}

func (p *pagePayloads) release(key payloadKey) {
	p.bytes -= len(p.bodies[key])
	delete(p.bodies, key)
	for _, quoted := range []bool{false, true} {
		view := prefixKey{key, quoted}
		p.bytes -= len(p.prefixes[view].text)
		delete(p.prefixes, view)
	}
}
