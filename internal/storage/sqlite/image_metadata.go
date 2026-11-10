// Purpose: Read an attachment's immutable metadata without restoring encoded pixels or message text.
// Pattern: Narrow repository projection.
// Usage: Authenticated thumbnail reads validate their source before consulting derived-byte caches.
// Related: internal/storage/sqlite/history_image.go, internal/commands/server/history_image.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/media"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"os"
)

func (s *Store) HistoryImageMetadata(ctx context.Context, sequence int64, index int) (media.Metadata, string, error) {
	var body []byte
	path := fmt.Sprintf("$.images[%d]", index)
	err := s.db.QueryRowContext(ctx, "SELECT json_extract(m.body,?) FROM timeline t JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence WHERE t.sequence=? AND m.role='user'", path, sequence).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return media.Metadata{}, "", fault.New(diag.NotFound, diag.ServerImageMissing)
	}
	if err != nil {
		return media.Metadata{}, "", err
	}
	if len(body) == 0 || string(body) == "null" {
		return media.Metadata{}, "", fault.New(diag.NotFound, diag.ServerImageMissing)
	}
	var value media.Metadata
	if err := json.Unmarshal(body, &value); err != nil {
		return value, "", err
	}
	if value.Width < 1 || value.Height < 1 || value.Width > media.MaxPixels || value.Height > media.MaxPixels || int64(value.Width)*int64(value.Height) > media.MaxPixels || len(value.Digest) != 64 {
		return value, "", fault.New(diag.IntegrityFailed, diag.ServerImageMissing)
	}
	info, err := os.Lstat(s.path)
	if err != nil {
		return value, "", err
	}
	stamp, err := conversationStamp(s.path, info)
	return value, stamp, err
}
