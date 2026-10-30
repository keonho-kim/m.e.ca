// Purpose: Restore recent user input text without hydrating image snapshots or tool output.
// Pattern: Consumer-specific repository projection.
// Usage: Terminal session restoration requests at most 200 editable input-history entries.
// Related: internal/tui/session/session.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"
	"fmt"
	"slices"

	json "github.com/goccy/go-json"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

// RecentInputs returns chronological user text, keeping binary and provider state out of the editor.
func (s *Store) RecentInputs(ctx context.Context, session string, limit int) ([]string, error) {
	if limit < 1 || limit > 200 {
		return nil, fmt.Errorf("input history limit must be between 1 and 200")
	}
	page, err := s.recordPage(ctx, `SELECT sequence,'message',body,refs,raw_bytes FROM messages
 WHERE session_id=? AND role='user' ORDER BY sequence DESC LIMIT ?`, session, limit)
	if err != nil {
		return nil, err
	}
	var inputs []string
	bytes := 0
	fields := map[string]bool{"content": true}
	for _, record := range page {
		body, err := s.unpackRecord(ctx, record, fields)
		if err != nil {
			return nil, err
		}
		bytes += len(body)
		if bytes > recordcontent.MaxBytes {
			break
		}
		var message struct{ Content string }
		if err := json.Unmarshal(body, &message); err != nil {
			return nil, err
		}
		inputs = append(inputs, message.Content)
	}
	slices.Reverse(inputs)
	return inputs, nil
}
