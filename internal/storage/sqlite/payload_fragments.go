// Purpose: Store shared text fragments once and reconstruct authenticated JSON strings.
// Pattern: Content-addressed manifest inside the existing payload table.
// Usage: LAT input packing writes manifests; payload hydration restores their exact strings.
// Related: internal/storage/sqlite/payload.go, internal/storage/sqlite/lat_input.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/storage/content"
)

const textFragmentsCodec = 3

func (s *Store) putTextFragments(ctx context.Context, tx *sql.Tx, raw []byte, parts []string) (string, error) {
	id := content.ID(raw)
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM payloads WHERE id=?", id).Scan(&exists); err != nil {
		return "", err
	}
	if exists > 0 {
		return id, nil
	}
	worthwhile, err := s.fragmentSavings(raw, parts)
	if err != nil {
		return "", err
	}
	if !worthwhile {
		return s.putPayload(ctx, tx, raw)
	}
	ids := make([]string, len(parts))
	for i, part := range parts {
		value, err := json.Marshal(part)
		if err != nil {
			return "", err
		}
		ids[i], err = s.putPayload(ctx, tx, value)
		if err != nil {
			return "", err
		}
	}
	body, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO payloads(id,codec,raw_bytes,body) VALUES(?,?,?,?)", id, textFragmentsCodec, len(raw), body)
	return id, err
}

// Compare the four phase inputs with one shared evidence value. Highly
// compressible inputs stay whole when fragment metadata would cost more.
func (s *Store) fragmentSavings(raw []byte, parts []string) (bool, error) {
	storedSize := func(value []byte) (int, error) {
		if len(value) <= content.Threshold {
			return len(value), nil
		}
		encoded, err := s.codec.Encode(value)
		return min(len(value), len(encoded)), err
	}
	whole, err := storedSize(raw)
	if err != nil {
		return false, err
	}
	cost := 4 * 256
	for i, part := range parts {
		encoded, err := json.Marshal(part)
		if err != nil {
			return false, err
		}
		size, err := storedSize(encoded)
		if err != nil {
			return false, err
		}
		if i == 1 {
			cost += size
		} else {
			cost += 4 * size
		}
	}
	return cost < 4*whole, nil
}

func (s *Store) textFragments(ctx context.Context, id string, size int, body []byte) ([]byte, error) {
	var parts []string
	if err := json.Unmarshal(body, &parts); err != nil {
		return nil, err
	}
	if len(parts) != 3 || size < 1 || size > content.MaxBytes {
		return nil, fmt.Errorf("invalid text fragment manifest")
	}
	var text strings.Builder
	for _, part := range parts {
		var codec, length int
		var stored []byte
		if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,body FROM payloads WHERE id=?", part).Scan(&codec, &length, &stored); err != nil {
			return nil, err
		}
		// Fragments are raw values, never another manifest. This prevents cycles
		// and bounds expansion before allocating the reconstructed source.
		if codec == textFragmentsCodec || length > size {
			return nil, fmt.Errorf("invalid nested or oversized text fragment")
		}
		raw, err := s.codec.Restore(codec, length, part, stored)
		if err != nil {
			return nil, fmt.Errorf("restore text fragment %s (%d bytes, codec %d): %w", part, length, codec, err)
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		if len(value) > size-text.Len() {
			return nil, fmt.Errorf("oversized text fragment expansion")
		}
		text.WriteString(value)
	}
	raw, err := json.Marshal(text.String())
	if err != nil {
		return nil, err
	}
	return s.codec.Restore(0, size, id, raw)
}
