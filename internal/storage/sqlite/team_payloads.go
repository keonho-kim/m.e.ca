// Purpose: Prepare and restore received public message bodies through authenticated room references.
// Pattern: Cross-session immutable payload reference.
// Usage: Private context commits create codec-3 references; normal hydration verifies the original body hash.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/payload.go, internal/agent/team/channel/context.go.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"strconv"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

type roomPayload struct {
	Sequence int64 `json:"sequence"`
}

func (p *journalEncoder) packMessage(message modelapi.Message) ([]byte, map[string]string, int, error) {
	s := p.store
	sequence, shared := channel.Sequence(message)
	// References have fixed envelope/index overhead; short text is smaller inline.
	if !shared || s.room == nil || message.Role == "tool" || len(message.Content) < 256 {
		body, err := json.Marshal(message)
		if err != nil {
			return nil, nil, 0, err
		}
		packed, refs, err := p.pack(body)
		return packed, refs, len(body), err
	}
	if message.Source.Session != "" && message.Source.Session != s.room.session.ID {
		return nil, nil, 0, fmt.Errorf("public message belongs to a different room")
	}
	encoded, err := s.room.teamContent.encoding(sequence, message.Content)
	if err != nil {
		return nil, nil, 0, err
	}
	id := encoded.id
	reference, _ := json.Marshal(roomPayload{Sequence: sequence})
	slot := strconv.Itoa(len(p.referenced))
	p.referenced = append(p.referenced, &preparedPayload{storedValue: storedValue{id: id, codec: 3, size: encoded.size, body: reference}})
	p.retained += len(reference)
	// Marshal metadata once; never allocate a full JSON message just to discard its content.
	type metadata modelapi.Message
	body, err := json.Marshal(struct {
		metadata
		Content any `json:"content"`
	}{metadata: metadata(message)})
	if err != nil {
		return nil, nil, 0, err
	}
	return body, map[string]string{`["content"]`: slot}, len(body) - len("null") + encoded.size, nil
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
