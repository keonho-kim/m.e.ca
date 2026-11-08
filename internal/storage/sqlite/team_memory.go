// Purpose: Retain authored collaboration content by public source reference for compaction.
// Pattern: Existing memory-column projection with batched source hydration.
// Usage: Event commits write reference metadata; MemoryRecords restores the published text.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/memory_sources.go.
package sqlite

import (
	"context"

	record "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) publicMemoryReference(ctx context.Context, event record.Event) (int64, int, error) {
	if event.Source == nil {
		return 0, 0, nil
	}
	if s.room == nil {
		return 0, 0, faults.Errorf(diagcatalog.SqlitePublicSourceRequiresItsOwningRoom)
	}
	origin := event.Source
	if origin.Session != "" && origin.Session != s.room.session.ID {
		return 0, 0, faults.Errorf(diagcatalog.SqlitePublicSourceBelongsToADifferentRoom)
	}
	if origin.Sequence < 1 {
		return 0, 0, faults.Errorf(diagcatalog.SqlitePublicSourceRequiresACommittedSequence)
	}

	meta, err := s.room.TeamMessageMetadata(ctx, origin.Sequence)
	if err != nil {
		return 0, 0, err
	}
	if meta.Kind != origin.Kind || meta.From != origin.Author {
		return 0, 0, faults.Errorf(diagcatalog.SqliteCollaborationOutcomeReferencesADifferentPublicRecord)
	}
	return origin.Sequence, meta.Bytes, nil
}

func (s *Store) publicContexts(ctx context.Context, sequences []any) (map[int64]string, error) {
	if s.room == nil {
		return nil, faults.Errorf(diagcatalog.SqlitePublicConversationOwnerIsRequired)
	}
	query := "SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes) " +
		"FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence " +
		"LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence " +
		"WHERE t.sequence IN (" + placeholders(len(sequences)) + ")"
	records, err := s.room.recordPage(ctx, query, sequences...)
	if err != nil {
		return nil, err
	}
	prefixes, err := s.room.publicPrefixes(ctx, records)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(records))
	for _, r := range records {
		m, err := s.room.publicMessage(ctx, r, prefixes[r.sequence])
		if err != nil {
			return nil, err
		}
		out[m.Sequence] = m.Context().Content
	}
	return out, nil
}
