// Purpose: Find unfinished native calls without restoring completed arguments or message images.
// Pattern: Paged metadata scan followed by selective payload hydration.
// Usage: Interrupted-work recovery consumes PendingTools in original call order.
// Related: internal/storage/recovery/recovery.go, internal/storage/sqlite/pending_projection.go.
package sqlite

import (
	"context"
	"fmt"

	json "github.com/goccy/go-json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type pendingToolGroup struct {
	record  storedRecord
	indices []int
}

// PendingTools loads full arguments only for calls without a subsequent result.
func (s *Store) PendingTools(ctx context.Context, session string) ([]toolapi.Call, error) {
	groups, err := s.pendingToolGroups(ctx, session)
	if err != nil {
		return nil, err
	}
	var calls []toolapi.Call
	for index := len(groups) - 1; index >= 0; index-- {
		group := groups[index]
		record, err := group.selectedRecord()
		if err != nil {
			return nil, err
		}
		body, err := s.unpackRecord(ctx, record, nil)
		if err != nil {
			return nil, err
		}
		var message modelapi.Message
		if err := json.Unmarshal(body, &message); err != nil {
			return nil, err
		}
		var selected []toolapi.Call
		for _, position := range group.indices {
			selected = append(selected, message.Calls[position])
		}
		if err := (modelapi.Message{Role: "assistant", Calls: selected}).Check(); err != nil {
			return nil, err
		}
		calls = append(calls, selected...)
	}
	return calls, nil
}

func (s *Store) pendingToolGroups(ctx context.Context, session string) ([]pendingToolGroup, error) {
	resolved := map[string]bool{}
	var groups []pendingToolGroup
	before := int64(1 << 62)
	for {
		page, err := s.recordPage(ctx, `SELECT sequence,'message',body,refs,raw_bytes FROM messages
 WHERE session_id=? AND sequence<? AND role IN ('assistant','tool') ORDER BY sequence DESC LIMIT 64`, session, before)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return groups, nil
		}
		for _, record := range page {
			// Packed metadata retains identities while large argument/image values
			// remain references. Matching a result never needs their original bytes.
			var header struct {
				Role       string
				Calls      []struct{ ID string }
				ToolCallID string `json:"tool_call_id"`
			}
			if err := json.Unmarshal(record.body, &header); err != nil {
				return nil, err
			}
			before = record.sequence
			if header.Role == "tool" {
				if header.ToolCallID == "" {
					return nil, fmt.Errorf("stored tool result lacks its call identity")
				}
				resolved[header.ToolCallID] = true
				continue
			}
			group := pendingToolGroup{record: record}
			for index, call := range header.Calls {
				if !resolved[call.ID] {
					group.indices = append(group.indices, index)
				}
				delete(resolved, call.ID)
			}
			if len(group.indices) > 0 {
				groups = append(groups, group)
			}
		}
	}
}
