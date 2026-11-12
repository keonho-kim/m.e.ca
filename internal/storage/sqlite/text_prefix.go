// Purpose: Read authenticated text previews without restoring an entire compressed string.
// Pattern: SQLite adapter for bounded content decoding.
// Usage: TimelineSummaries reserves visible bytes and a scoped decoder budget before streaming payloads.
// Related: internal/storage/content/prefix_reader.go, internal/storage/sqlite/read_budget.go.
package sqlite

import (
	"context"
	"encoding/hex"
	"io"

	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) readTextPrefix(ctx context.Context, id string, budget *readBudget, limit int) (string, bool, error) {
	id, quoted := content.Reference(id)
	var value admittedPayload
	var stored int
	err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,length(body),digest FROM payloads WHERE id=?", id).Scan(&value.codec, &value.size, &stored, &value.digest)
	if err != nil {
		return "", false, err
	}
	if value.codec == 3 {
		value, err = s.admitPayload(ctx, id, budget)
		if err != nil {
			return "", false, err
		}
		raw, err := s.restorePayload(ctx, value.codec, value.size, value.digest, value.body)
		if err != nil {
			return "", false, err
		}
		return content.StringPrefix(raw, limit)
	}
	// Encoded text cannot decode to more bytes than its authenticated raw size.
	// Small prefixes must not reserve the caller's entire remaining window.
	limit = min(limit, value.size)
	if !budget.previewScratch {
		budget.scratch = content.PreviewScratchBytes
		budget.previewScratch = true
	}
	// The validator may retain six encoded bytes per visible byte (\u0000).
	// Charge its actual raw-size bound, rather than assuming an ASCII prefix.
	if err := budget.take(min(value.size, 6*limit+16)); err != nil {
		return "", false, err
	}
	reader := &payloadChunks{ctx: ctx, store: s, id: id, size: stored}
	memory := processmemory.NewPayloadLease()
	defer memory.Close()
	return content.StreamTextPrefix(ctx, value.codec, value.size, hex.EncodeToString(value.digest), reader, limit, quoted, memory.Resize)
}

type payloadChunks struct {
	ctx          context.Context
	store        *Store
	id           string
	size, offset int
	buffer       []byte
}

func (r *payloadChunks) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if len(r.buffer) == 0 {
		if r.offset >= r.size {
			return 0, io.EOF
		}
		if err := r.store.db.QueryRowContext(r.ctx, "SELECT substr(body,?,?) FROM payloads WHERE id=?", r.offset+1, min(64<<10, r.size-r.offset), r.id).Scan(&r.buffer); err != nil {
			return 0, err
		}
		if len(r.buffer) == 0 {
			return 0, io.ErrUnexpectedEOF
		}
		r.offset += len(r.buffer)
	}
	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}
