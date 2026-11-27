// Purpose: Restore composed JSON with bounded leaf reads and end-to-end integrity checks.
// Pattern: SQLite reference adapter without recursive composition or cross-session access.
// Usage: Payload and preview readers restore codec-5 values through this owner.
// Related: internal/storage/sqlite/payload_composition.go, internal/storage/sqlite/payload.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) restoreComposition(ctx context.Context, size int, digest, body []byte) (json.RawMessage, error) {
	invalid := func() error { return faults.Errorf(diag.ContentStoredPayloadIntegrityCheckFailed) }
	if size < 1 || size > content.MaxBytes || len(body) > size {
		return nil, invalid()
	}
	var parts []payloadPart
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('[') {
		return nil, invalid()
	}
	for decoder.More() {
		if len(parts) >= 2*maxCompositionReferences+1 {
			return nil, invalid()
		}
		var part payloadPart
		if err := decoder.Decode(&part); err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, invalid()
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, invalid()
	}
	uses := make(map[int64]int)
	for _, part := range parts {
		if (part.Reference > 0) == (len(part.Literal) > 0) || part.Reference < 0 {
			return nil, invalid()
		}
		if part.Reference > 0 {
			uses[part.Reference]++
		}
	}
	// Repeated references in one result reuse their decoded bytes until the last
	// occurrence; the existing page reuse budget bounds retained originals.
	cache := make(map[int64][]byte)
	retained := 0
	result := make([]byte, 0, size)
	for _, part := range parts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value := part.Literal
		if part.Reference > 0 {
			value = cache[part.Reference]
			if value == nil {
				var err error
				value, err = s.compositionLeaf(ctx, part.Reference, size-len(result))
				if err != nil {
					return nil, err
				}
				if uses[part.Reference] > 1 && len(value) <= pagePayloadBytes-retained {
					cache[part.Reference] = value
					retained += len(value)
				}
			}
			uses[part.Reference]--
			if uses[part.Reference] == 0 {
				retained -= len(cache[part.Reference])
				delete(cache, part.Reference)
			}
		}
		if len(value) > size-len(result) {
			return nil, invalid()
		}
		result = append(result, value...)
	}
	return s.codec.Restore(0, size, hex.EncodeToString(digest), result)
}

func (s *Store) compositionLeaf(ctx context.Context, id int64, maximum int) ([]byte, error) {
	var codec, size, stored int
	var digest, body []byte
	err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,length(body),digest FROM payloads WHERE id=?", id).Scan(&codec, &size, &stored, &digest)
	if err != nil {
		return nil, err
	}
	// Only independent immutable leaf codecs are allowed. A corrupted reference
	// cannot form a cycle, restore another session or inflate beyond its parent.
	if codec < 0 || codec > 2 || size < 1 || size > maximum || stored < 1 || stored > size {
		return nil, faults.Errorf(diag.ContentStoredPayloadIntegrityCheckFailed)
	}
	err = s.db.QueryRowContext(ctx, "SELECT body FROM payloads WHERE id=? AND codec=? AND raw_bytes=? AND length(body)=? AND digest=?", strconv.FormatInt(id, 10), codec, size, stored, digest).Scan(&body)
	if err != nil {
		return nil, err
	}
	return s.codec.Restore(codec, size, hex.EncodeToString(digest), body)
}
