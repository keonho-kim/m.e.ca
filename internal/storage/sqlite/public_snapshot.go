// Purpose: Freeze and authenticate public attribution separately from its immutable original text.
// Pattern: Transactional prefix snapshot with page-scoped batch reads.
// Usage: Public append writes one snapshot; room and compaction readers restore it before projection.
// Related: internal/storage/sqlite/team_messages.go, internal/storage/sqlite/team_memory.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) writePublicSnapshot(ctx context.Context, tx *sql.Tx, message *channel.Message, sequence int64) error {
	message.Sequence, message.Session = sequence, s.session.ID
	message.Prefix = nil
	prefix := message.ContextPrefix()
	raw, err := json.Marshal(prefix)
	if err != nil {
		return err
	}
	encoded, err := s.encodeValue(content.ID(raw), raw)
	if err != nil {
		return err
	}
	digest, err := hex.DecodeString(encoded.id)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO public_contexts(sequence,prefix,codec,raw_bytes,digest) VALUES(?,?,?,?,?)", sequence, encoded.body, encoded.codec, encoded.size, digest); err != nil {
		return err
	}
	message.Prefix = &prefix
	return nil
}

func (s *Store) publicPrefixes(ctx context.Context, records []storedRecord) (prefixes map[int64]*string, resultErr error) {
	if len(records) == 0 {
		return nil, nil
	}
	limit := 0
	bounds := make(map[int64]int, len(records))
	args := make([]any, 2, len(records)+2)
	for _, r := range records {
		bounds[r.sequence] = min(content.MaxBytes, r.size+1024)
		limit = max(limit, bounds[r.sequence])
		args = append(args, r.sequence)
	}
	args[0], args[1] = limit, limit
	rows, err := s.db.QueryContext(ctx, "SELECT sequence,CASE WHEN raw_bytes BETWEEN 1 AND ? AND length(prefix)<=? THEN prefix END,codec,raw_bytes,digest FROM public_contexts WHERE sequence IN ("+placeholders(len(records))+")", args...)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, rows.Close()) }()
	prefixes = make(map[int64]*string)
	for rows.Next() {
		var sequence int64
		var codec, size int
		var body, digest []byte
		if err := rows.Scan(&sequence, &body, &codec, &size, &digest); err != nil {
			return nil, err
		}
		if len(body) == 0 || size > bounds[sequence] || len(body) > bounds[sequence] || len(digest) != 32 {
			return nil, fmt.Errorf("invalid public context prefix length")
		}
		prefix, _, err := s.codec.TextPrefix(ctx, codec, size, hex.EncodeToString(digest), body, size)
		if err != nil {
			return nil, fmt.Errorf("public context prefix failed integrity verification: %w", err)
		}
		prefixes[sequence] = &prefix
	}
	return prefixes, rows.Err()
}
