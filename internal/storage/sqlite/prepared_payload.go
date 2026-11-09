// Purpose: Encode immutable payloads before a write transaction and recheck their state inside it.
// Pattern: Transaction-bound prepared values with group-local deduplication.
// Usage: Journal preparation packs references; record application inserts each prepared payload once.
// Related: internal/storage/sqlite/prepare.go, internal/storage/content/pack.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"

	"strconv"

	json "github.com/goccy/go-json"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

type storedValue struct {
	id          string
	codec, size int
	body        []byte
}
type preparedPayload struct {
	storedValue
	known, written bool
	local          int64
	digest         []byte
}

func (s *Store) encodeValue(id string, body []byte) (storedValue, error) {
	value := storedValue{id: id, size: len(body), body: body}
	if len(body) > content.Threshold {
		encoded, err := s.codec.Encode(body)
		if err != nil {
			return storedValue{}, err
		}
		if len(encoded) < len(body) {
			value.codec, value.body = 2, encoded
		}
	}
	return value, nil
}

func (p *journalEncoder) pack(body []byte, shared string) ([]byte, map[string]string, error) {
	return content.Pack(body, shared, p.payload)
}

func (p *journalEncoder) payload(raw []byte) (string, error) {
	id := content.ID(raw)
	value := p.payloads[id]
	if value == nil {
		value = &preparedPayload{storedValue: storedValue{id: id, size: len(raw)}}
		digest, err := hex.DecodeString(id)
		if err != nil {
			return "", err
		}
		value.digest = digest
		err = p.lookup.QueryRowContext(p.ctx, "SELECT id FROM payloads WHERE digest=?", digest).Scan(&value.local)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		value.known = err == nil

		if !value.known {
			encoded, err := p.store.encodeValue(id, raw)
			if err != nil {
				return "", err
			}
			value.storedValue = encoded
		}
		if p.payloads == nil {
			p.payloads = make(map[string]*preparedPayload)
		}
		p.payloads[id] = value
		// Include conservative allocation overhead for the value and its map slot.
		p.retained += 256 + len(id) + len(value.body)
	}
	p.referenced = append(p.referenced, value)
	return strconv.Itoa(len(p.referenced) - 1), nil
}

func (p *preparedPayload) write(ctx context.Context, tx *sql.Tx) error {
	if p.written {
		return nil
	}
	if p.digest == nil {
		var err error
		p.digest, err = hex.DecodeString(p.id)
		if err != nil {
			return err
		}
	}
	if !p.known {
		result, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO payloads(digest,codec,raw_bytes,body) VALUES(?,?,?,?)", p.digest, p.codec, p.size, p.body)
		if err != nil {
			return err
		}
		inserted, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if inserted == 1 {
			p.local, err = result.LastInsertId()
			if err != nil {
				return err
			}
		}
	}
	if p.known || p.local == 0 {
		if err := tx.QueryRowContext(ctx, "SELECT id FROM payloads WHERE digest=?", p.digest).Scan(&p.local); err != nil {
			return faults.Errorf(diagcatalog.SqlitePreparedImmutablePayloadUnavailable, err)
		}
	}
	p.written, p.body = true, nil
	return nil
}

func (v *preparedRecord) localRefs() error {
	if len(v.referenceSlots) == 0 {
		v.refs = []byte("{}")
		return nil
	}
	for path, slot := range v.referenceSlots {
		slot, quote := content.Reference(slot)
		index, err := strconv.Atoi(slot)
		if err != nil || index < 0 || index >= len(v.payloads) || v.payloads[index].local < 1 {
			return faults.Errorf(diagcatalog.SqliteUnresolvedPreparedPayloadReference)
		}
		v.referenceSlots[path] = strconv.FormatInt(v.payloads[index].local, 10)
		if quote {
			v.referenceSlots[path] = content.StringReference(v.referenceSlots[path])
		}
	}
	body, err := json.Marshal(v.referenceSlots)
	v.refs, v.referenceSlots = body, nil
	return err
}
