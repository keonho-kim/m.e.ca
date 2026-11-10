// Purpose: Restore public image references directly into private model values without a JSON round trip.
// Pattern: Typed repository hydration with canonical reference verification.
// Usage: ContextMessages selects this path only for complete image-bearing private records.
// Related: internal/storage/sqlite/context_record.go, internal/storage/sqlite/team_images.go.
package sqlite

import (
	"context"
	"encoding/hex"
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) roomImageContext(ctx context.Context, r storedRecord) (modelapi.Message, bool, error) {
	var refs map[string]string
	if len(r.refs) == 0 {
		return modelapi.Message{}, false, nil
	}
	if err := json.Unmarshal(r.refs, &refs); err != nil {
		return modelapi.Message{}, false, err
	}
	id := refs[`["images"]`]
	if id == "" {
		return modelapi.Message{}, false, nil
	}
	var value admittedPayload
	if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,digest,body FROM payloads WHERE id=?", id).Scan(&value.codec, &value.size, &value.digest, &value.body); err != nil {
		return modelapi.Message{}, false, err
	}
	if value.codec != 4 {
		return modelapi.Message{}, false, nil
	}
	images, err := s.verifiedRoomImages(ctx, hex.EncodeToString(value.digest), value.size, value.body)
	if err != nil {
		return modelapi.Message{}, true, err
	}
	delete(refs, `["images"]`)
	remaining, _ := json.Marshal(refs)
	body, err := content.Restore(r.body, remaining, nil, func(id string) (json.RawMessage, error) { return s.payload(ctx, id) })
	if err != nil {
		return modelapi.Message{}, true, err
	}
	var message modelapi.Message
	if err := json.Unmarshal(body, &message); err != nil {
		return message, true, err
	}
	var placeholder struct{ Images json.RawMessage }
	if json.Unmarshal(body, &placeholder) != nil || string(placeholder.Images) != "null" {
		return message, true, faults.Errorf(diagcatalog.SqliteInvalidSharedContentPlaceholder)
	}
	message.Images = images
	return message, true, nil
}
