// Purpose: Restore one explicitly requested user attachment from its owning session journal.
// Pattern: Bounded repository read with image integrity verification.
// Usage: Authenticated image downloads supply a restoration memory reservation.
// Related: internal/storage/sqlite/timeline.go, internal/commands/server/history_image.go.
package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) HistoryImage(ctx context.Context, sequence int64, index int, bounds history.ReadBounds) (media.Image, error) {
	var r storedRecord
	err := s.db.QueryRowContext(ctx, "SELECT m.body,m.refs,m.raw_bytes FROM timeline t JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence WHERE t.sequence=? AND m.role='user'", sequence).Scan(&r.body, &r.refs, &r.size)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return media.Image{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerImageMissing)
		}
		return media.Image{}, err
	}
	var envelope struct{ Images []json.RawMessage }
	if err := json.Unmarshal(r.body, &envelope); err != nil {
		return media.Image{}, err
	}
	if index < 0 || index >= len(envelope.Images) {
		return media.Image{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerImageMissing)
	}
	var refs map[string]string
	if len(r.refs) > 0 {
		if err := json.Unmarshal(r.refs, &refs); err != nil {
			return media.Image{}, err
		}
	}
	selected := make(map[string]string)
	for path, id := range refs {
		var parts []string
		if json.Unmarshal([]byte(path), &parts) != nil {
			return media.Image{}, faults.New(diagcatalog.IntegrityFailed, diagcatalog.ServerImageMissing)
		}
		if len(parts) >= 3 && parts[0] == "images" && parts[1] == strconv.Itoa(index) {
			key, _ := json.Marshal(parts[2:])
			selected[string(key)] = id
		}
	}
	references, _ := json.Marshal(selected)
	budget := newReadBudget(bounds.MaxBytes, func(n int) error {
		if bounds.Reserve == nil {
			return nil
		}
		return bounds.Reserve(3*n + len(r.body))
	})
	body, err := content.Restore(envelope.Images[index], references, nil, func(id string) (json.RawMessage, error) { return s.readPayload(ctx, id, budget) })
	if err != nil {
		return media.Image{}, err
	}
	var image media.Image
	if err := json.Unmarshal(body, &image); err != nil {
		return media.Image{}, err
	}
	if err := image.Check(); err != nil {
		return media.Image{}, err
	}
	digest := sha256.Sum256(image.Data)
	if hex.EncodeToString(digest[:]) != image.Digest {
		return media.Image{}, faults.New(diagcatalog.IntegrityFailed, diagcatalog.ContentStoredPayloadIntegrityCheckFailed)
	}
	return image, nil
}
