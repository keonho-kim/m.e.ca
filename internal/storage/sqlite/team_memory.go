// Purpose: Retain authored collaboration content by public source reference for compaction.
// Pattern: Existing memory-column projection with batched source hydration.
// Usage: Event commits write reference metadata; MemoryRecords restores the published text.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/memory_sources.go.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	record "github.com/keonho-kim/meca/internal/agent/record"
	"strconv"
	"strings"
)

func (s *Store) publicMemoryReference(ctx context.Context, event record.Event) (string, int, error) {
	if s.room == nil || event.Kind != "tool_succeeded" || event.Call == nil {
		return "", 0, nil
	}
	kind := ""
	switch event.Call.Name {
	case "discuss":
		kind = "team_discussion"
	case "set_work":
		kind = "team_work"
	default:
		return "", 0, nil
	}
	var result struct{ Sequence int64 }
	if err := json.Unmarshal(event.Output, &result); err != nil || result.Sequence < 1 {
		return "", 0, fmt.Errorf("successful collaboration outcome requires its public sequence")
	}
	meta, err := s.room.TeamMessageMetadata(ctx, result.Sequence)
	if err != nil {
		return "", 0, err
	}
	if meta.Kind != kind {
		return "", 0, fmt.Errorf("collaboration outcome references a different public record kind")
	}
	return "public:" + strconv.FormatInt(result.Sequence, 10), meta.Bytes, nil
}

func publicMemorySequence(role string) (int64, error) {
	if !strings.HasPrefix(role, "public:") {
		return 0, nil
	}
	sequence, err := strconv.ParseInt(strings.TrimPrefix(role, "public:"), 10, 64)
	if err != nil || sequence < 1 {
		return 0, fmt.Errorf("invalid authored public reference")
	}
	return sequence, nil
}

func (s *Store) publicContexts(ctx context.Context, sequences []any) (map[int64]string, error) {
	if s.room == nil {
		return nil, fmt.Errorf("public conversation owner is required")
	}
	query := "SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes) " +
		"FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence " +
		"LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence " +
		"WHERE t.session_id=? AND t.sequence IN (" + placeholders(len(sequences)) + ")"
	records, err := s.room.recordPage(ctx, query, append([]any{s.room.session.ID}, sequences...)...)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(records))
	for _, r := range records {
		m, err := s.room.publicMessage(ctx, r)
		if err != nil {
			return nil, err
		}
		out[m.Sequence] = m.Context().Content
	}
	return out, nil
}
