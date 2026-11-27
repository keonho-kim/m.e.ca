// Purpose: Share existing native JSON originals inside a code result without copying their stored bodies.
// Pattern: Lossless payload composition using verified local leaf references.
// Usage: Journal preparation encodes code envelopes; ordinary payload reads restore them.
// Related: internal/storage/sqlite/prepared_payload.go, internal/storage/sqlite/payload_composition_read.go.
package sqlite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

const composedPayloadCodec = 5
const maxCompositionReferences = 128

type payloadPart struct {
	Literal   []byte `json:"literal,omitempty"`
	Reference int64  `json:"reference,omitempty"`
}

type payloadRegion struct {
	start, end int
	dependency payloadDependency
}

type payloadDependency struct {
	reference int64
	size      int
	digest    [32]byte
}

func (p *journalEncoder) composeValue(id string, raw []byte) (storedValue, bool, error) {
	if len(raw) <= content.Threshold || raw[0] != '{' {
		return storedValue{}, false, nil
	}
	var regions []payloadRegion
	remaining := maxCompositionReferences
	var inspect func([]byte, int, int) error
	inspect = func(value []byte, offset, depth int) error {
		if len(value) <= content.Threshold || remaining == 0 || depth > 8 {
			return nil
		}
		if err := p.ctx.Err(); err != nil {
			return err
		}
		if depth > 0 {
			remaining--
			digest := sha256.Sum256(value)
			var reference int64
			err := p.lookup.QueryRowContext(p.ctx, "SELECT id FROM payloads WHERE digest=? AND raw_bytes=? AND codec IN (0,1,2) AND length(body)<=raw_bytes", digest[:], len(value)).Scan(&reference)
			if err == nil {
				regions = append(regions, payloadRegion{offset, offset + len(value), payloadDependency{reference, len(value), digest}})
				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		if value[0] != '{' && value[0] != '[' {
			return nil
		}
		decoder := json.NewDecoder(bytes.NewReader(value))
		opening, err := decoder.Token()
		if err != nil {
			return err
		}
		for decoder.More() {
			if opening == json.Delim('{') {
				if _, err := decoder.Token(); err != nil {
					return err
				}
			}
			var child json.RawMessage
			if err := decoder.Decode(&child); err != nil {
				return err
			}
			start := int(decoder.InputOffset()) - len(child)
			if err := inspect(child, offset+start, depth+1); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	}
	if err := inspect(raw, 0, 0); err != nil {
		return storedValue{}, false, err
	}
	if len(regions) == 0 {
		return storedValue{}, false, nil
	}
	parts := make([]payloadPart, 0, len(regions)*2+1)
	dependencies := make([]payloadDependency, 0, len(regions))
	seen := make(map[int64]bool)
	previous := 0
	for _, region := range regions {
		if region.start > previous {
			parts = append(parts, payloadPart{Literal: raw[previous:region.start]})
		}
		parts = append(parts, payloadPart{Reference: region.dependency.reference})
		if !seen[region.dependency.reference] {
			dependencies = append(dependencies, region.dependency)
			seen[region.dependency.reference] = true
		}
		previous = region.end
	}
	if previous < len(raw) {
		parts = append(parts, payloadPart{Literal: raw[previous:]})
	}
	body, err := json.Marshal(parts)
	if err != nil {
		return storedValue{}, false, err
	}
	if len(body) >= len(raw) {
		return storedValue{}, false, nil
	}
	return storedValue{id: id, codec: composedPayloadCodec, size: len(raw), body: body, dependencies: dependencies}, true, nil
}

func (p *preparedPayload) checkDependencies(ctx context.Context, tx *sql.Tx) error {
	for _, dependency := range p.dependencies {
		var id int64
		err := tx.QueryRowContext(ctx, "SELECT id FROM payloads WHERE id=? AND digest=? AND raw_bytes=? AND codec IN (0,1,2) AND length(body)<=raw_bytes", dependency.reference, dependency.digest[:], dependency.size).Scan(&id)
		if err != nil {
			return faults.Errorf(diag.SqlitePreparedImmutablePayloadUnavailable, err)
		}
	}
	return nil
}
