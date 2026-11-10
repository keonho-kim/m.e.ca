// Purpose: Prepare and restore received public message bodies through authenticated room references.
// Pattern: Cross-session immutable payload reference.
// Usage: Private context commits create codec-3 references; normal hydration verifies the original body hash.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/payload.go, internal/agent/team/channel/context.go.
package sqlite

import (
	"context"
	json "github.com/goccy/go-json"

	"strconv"

	"github.com/keonho-kim/meca/internal/agent/team/channel"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

type roomPayload struct {
	Sequence int64 `json:"sequence"`
}

func (p *journalEncoder) packMessage(message modelapi.Message) ([]byte, map[string]string, int, error) {
	s := p.store
	sequence, shared := channel.Sequence(message)
	if shared && s.room != nil && len(message.Images) > 0 {
		return p.packTeamImages(message, sequence)
	}
	// References have fixed envelope/index overhead; short text is smaller inline.
	if !shared || s.room == nil || message.Role == "tool" || len(message.Content) < 256 {
		body, err := json.MarshalWithOption(message, json.DisableHTMLEscape())
		if err != nil {
			return nil, nil, 0, err
		}
		if len(body) > recordcontent.MaxBytes {
			return nil, nil, 0, faults.Errorf(diagcatalog.ModelMessageExceedsSizeLimits)
		}
		sharedField := ""
		if len(message.Reasoning) >= recordcontent.SharedThreshold {
			sharedField = "reasoning"
		}
		packed, refs, err := p.pack(body, sharedField)
		return packed, refs, len(body), err
	}
	if message.Source.Session != "" && message.Source.Session != s.room.session.ID {
		return nil, nil, 0, faults.Errorf(diagcatalog.SqlitePublicMessageBelongsToADifferentRoom)
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
		return nil, faults.Errorf(diagcatalog.SqliteSharedConversationOwnerIsRequiredToRestore)
	}
	var reference roomPayload
	if err := json.Unmarshal(body, &reference); err != nil || reference.Sequence < 1 {
		return nil, faults.Errorf(diagcatalog.SqliteInvalidPublicConversationReference)
	}
	messages, err := s.room.TeamMessages(ctx, reference.Sequence-1, reference.Sequence, 1)
	if err != nil {
		return nil, err
	}
	if len(messages) != 1 || messages[0].Sequence != reference.Sequence {
		return nil, faults.Errorf(diagcatalog.SqliteReferencedPublicMessageIsMissing)
	}
	raw, err := json.Marshal(messages[0].Context().Content)
	if err != nil {
		return nil, err
	}
	if len(raw) != size || recordcontent.ID(raw) != id {
		return nil, faults.Errorf(diagcatalog.SqliteReferencedPublicMessageFailedLengthHashVerification)
	}
	return raw, nil
}
