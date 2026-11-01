// Purpose: Share the exact common evidence bytes in persisted LAT phase inputs.
// Pattern: Event-specific storage projection using content-addressed fragments.
// Usage: writeEvent supplies only canonical LAT user messages to packEvent.
// Related: internal/storage/sqlite/payload_fragments.go, internal/agent/lat/packet.go.
package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) packEvent(ctx context.Context, tx *sql.Tx, body []byte, event runrecord.Event) ([]byte, []byte, error) {
	if event.Kind != "lat_message" || event.LAT == nil || event.Message == nil || event.Message.Role != "user" || len(event.Message.Content) <= content.Threshold {
		return s.packRecord(ctx, tx, body)
	}
	parts := evidenceParts(event.Message.Content)
	if parts == nil {
		return s.packRecord(ctx, tx, body)
	}
	encoded, err := json.Marshal(event.Message.Content)
	if err != nil {
		return nil, nil, err
	}
	return content.Pack(body, func(raw []byte) (string, error) {
		if bytes.Equal(raw, encoded) {
			return s.putTextFragments(ctx, tx, raw, parts)
		}
		return s.putPayload(ctx, tx, raw)
	})
}

func evidenceParts(text string) []string {
	if !json.Valid([]byte(text)) {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil
	}
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			return nil
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil
		}
		if name == "independent_evidence" && len(value) >= 16*1024 {
			end := int(decoder.InputOffset())
			start := end - len(value)
			return []string{text[:start], text[start:end], text[end:]}
		}
	}
	return nil
}
