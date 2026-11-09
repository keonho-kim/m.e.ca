// Purpose: Store private image receives as verified references to the public room's immutable snapshots.
// Pattern: Cross-session payload codec owned by the SQLite adapter.
// Usage: packMessage uses codec 4 for public images; restoration verifies the full canonical image value.
// Related: internal/storage/sqlite/team_payloads.go, internal/storage/sqlite/payload.go.
package sqlite

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/media"
	"strconv"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (p *journalEncoder) packTeamImages(message modelapi.Message, sequence int64) ([]byte, map[string]string, int, error) {
	id, imageSize, err := teamImageIdentity(message.Images)
	if err != nil {
		return nil, nil, 0, err
	}
	if imageSize > content.MaxBytes {
		return nil, nil, 0, faults.Errorf(diagcatalog.ModelMessageExceedsSizeLimits)
	}
	message.Images = nil
	body, refs, size, err := p.packMessage(message)
	if err != nil {
		return nil, nil, 0, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, nil, 0, err
	}
	fields["images"] = json.RawMessage("null")
	packed, err := json.Marshal(fields)
	if err != nil {
		return nil, nil, 0, err
	}
	reference, _ := json.Marshal(roomPayload{Sequence: sequence})
	slot := strconv.Itoa(len(p.referenced))
	p.referenced = append(p.referenced, &preparedPayload{storedValue: storedValue{id: id, codec: 4, size: imageSize, body: reference}})
	p.retained += len(reference)
	if refs == nil {
		refs = make(map[string]string)
	}
	refs[`["images"]`] = slot
	return packed, refs, size + len(packed) - len(body) - 4 + imageSize, nil
}

func (s *Store) verifiedRoomImages(ctx context.Context, id string, size int, body []byte) ([]media.Image, error) {
	if s.room == nil {
		return nil, faults.Errorf(diagcatalog.SqliteSharedConversationOwnerIsRequiredToRestore)
	}
	var ref roomPayload
	if json.Unmarshal(body, &ref) != nil || ref.Sequence < 1 {
		return nil, faults.Errorf(diagcatalog.SqliteInvalidPublicConversationReference)
	}
	values, err := s.room.TeamMessages(ctx, ref.Sequence-1, ref.Sequence, 1)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 || values[0].Sequence != ref.Sequence || values[0].From != "user" {
		return nil, faults.Errorf(diagcatalog.SqliteReferencedPublicMessageIsMissing)
	}
	actual, actualSize, err := teamImageIdentity(values[0].Images)
	if err != nil {
		return nil, err
	}
	if actualSize != size || actual != id {
		return nil, faults.Errorf(diagcatalog.SqliteReferencedPublicMessageFailedLengthHashVerification)
	}
	return values[0].Images, nil
}

func (s *Store) roomImages(ctx context.Context, id string, size int, body []byte) (json.RawMessage, error) {
	images, err := s.verifiedRoomImages(ctx, id, size, body)
	if err != nil {
		return nil, err
	}
	return json.Marshal(images)
}
