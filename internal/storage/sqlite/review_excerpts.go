// Purpose: Authenticate repeated LAT payloads once per source page and reuse only small excerpts.
// Pattern: Last-use eviction with a page-owned bounded excerpt cache.
// Usage: LATRecords owns the index and cache; latObservation reads through excerpt.
// Related: internal/storage/sqlite/lat_observation.go, internal/storage/sqlite/page_references.go.
package sqlite

import "context"

type reviewExcerpt struct {
	body     []byte
	original int
}

type reviewExcerpts struct {
	store   *Store
	last    map[string]int
	entries map[string]reviewExcerpt
	index   int
	bytes   int
}

func (p *reviewExcerpts) excerpt(ctx context.Context, id string) ([]byte, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	value, cached := p.entries[id]
	if !cached {
		var codec int
		var body []byte
		if err := p.store.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,body FROM payloads WHERE id=?", id).Scan(&codec, &value.original, &body); err != nil {
			return nil, 0, err
		}
		var err error
		value.body, err = p.store.codec.Excerpt(ctx, codec, value.original, id, body, 8192)
		if err != nil {
			return nil, 0, err
		}
	}
	if p.last[id] <= p.index {
		if cached {
			p.bytes -= len(value.body)
			delete(p.entries, id)
		}
	} else if !cached && len(value.body) <= 256*1024-p.bytes {
		if p.entries == nil {
			p.entries = make(map[string]reviewExcerpt)
		}
		p.entries[id] = value
		p.bytes += len(value.body)
	}
	return value.body, value.original, nil
}
