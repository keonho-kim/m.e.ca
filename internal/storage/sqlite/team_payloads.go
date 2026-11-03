// Purpose: Store received public message bodies by authenticated room reference instead of receiver copies.
// Pattern: Cross-session immutable payload reference.
// Usage: Private context commits create codec-3 references; normal hydration verifies the original body hash.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/payload.go, internal/agent/team/channel/context.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

type roomPayload struct {
	Sequence int64 `json:"sequence"`
}

func (s *Store) packMessage(ctx context.Context, tx *sql.Tx, message modelapi.Message) ([]byte, []byte, int, error) {
	sequence, shared := channel.Sequence(message.ID)
	// References have fixed envelope/index overhead; short text is smaller inline.
	if !shared || s.room == nil || len(message.Content) < 256 {
		body, err := json.Marshal(message)
		if err != nil {
			return nil, nil, 0, err
		}
		packed, refs, err := s.packRecord(ctx, tx, body)
		return packed, refs, len(body), err
	}
	raw, err := json.Marshal(message.Content)
	if err != nil {
		return nil, nil, 0, err
	}
	id := recordcontent.ID(raw)
	reference, _ := json.Marshal(roomPayload{Sequence: sequence})
	if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO payloads(id,codec,raw_bytes,body) VALUES(?,3,?,?)", id, len(raw), reference); err != nil {
		return nil, nil, 0, err
	}
	// Marshal metadata once; never allocate a full JSON message just to discard its content.
	type metadata modelapi.Message
	body, err := json.Marshal(struct {
		metadata
		Content any `json:"content"`
	}{metadata: metadata(message)})
	if err != nil {
		return nil, nil, 0, err
	}
	refs, err := json.Marshal(map[string]string{`["content"]`: id})
	return body, refs, len(body) - len("null") + len(raw), err
}

func (s *Store) roomPayload(ctx context.Context, id string, size int, body []byte) (json.RawMessage, error) {
	if s.room == nil {
		return nil, fmt.Errorf("shared conversation owner is required to restore this private context")
	}
	var reference roomPayload
	if err := json.Unmarshal(body, &reference); err != nil || reference.Sequence < 1 {
		return nil, fmt.Errorf("invalid public conversation reference")
	}
	messages, err := s.room.TeamMessages(ctx, reference.Sequence-1, reference.Sequence, 1)
	if err != nil {
		return nil, err
	}
	if len(messages) != 1 || messages[0].Sequence != reference.Sequence {
		return nil, fmt.Errorf("referenced public message is missing")
	}
	raw, err := json.Marshal(messages[0].Context().Content)
	if err != nil {
		return nil, err
	}
	if len(raw) != size || recordcontent.ID(raw) != id {
		return nil, fmt.Errorf("referenced public message failed length/hash verification")
	}
	return raw, nil
}
