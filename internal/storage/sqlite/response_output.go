// Purpose: Read committed public assistant text by immutable task timeline boundaries.
// Pattern: Selective SQLite result projection.
// Usage: The server result reader opens a conversation without running or restoring it.
// Related: internal/commands/server/response.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"

	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) OutputPosition(ctx context.Context, session string) (int64, error) {
	if err := s.checkSession(session); err != nil {
		return 0, err
	}
	var position int64
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence),0) FROM timeline").Scan(&position)
	return position, err
}

func (s *Store) ResponseOutput(ctx context.Context, session string, after, through int64, maxBytes int) (string, error) {
	if err := s.checkSession(session); err != nil {
		return "", err
	}
	if after < 0 || through < after {
		return "", faults.Errorf(diagcatalog.SqliteInvalidOutputBoundaries)
	}
	var text strings.Builder
	budget := newReadBudget(maxBytes)
	for {
		page, err := s.outputPage(ctx, session, after, through, budget)
		if err != nil {
			return "", err
		}
		if len(page) == 0 {
			return text.String(), nil
		}
		for _, record := range page {
			if err := budget.take(len(record.body)); err != nil {
				return "", err
			}
			after = record.sequence
			body, err := content.Text(record.body, record.refs, func(id string) (json.RawMessage, error) { return s.readPayload(ctx, id, budget) })
			if err != nil {
				return "", err
			}
			body = strings.TrimSpace(body)
			if body == "" {
				continue
			}
			if text.Len() > 0 {
				text.WriteString("\n\n")
			}
			text.WriteString(body)
		}
	}
}

func (s *Store) outputPage(ctx context.Context, session string, after, through int64, budget *readBudget) ([]storedRecord, error) {
	maxBytes, count := content.MaxBytes, 16
	if budget.limited {
		maxBytes, count = budget.remaining, 1
	}
	if through-after <= 64 {
		// A short timeline range already bounds work, regardless of total history.
		return s.recordPage(ctx, "SELECT t.sequence,t.kind,CASE WHEN length(m.body)<=? THEN m.body END,m.refs,m.raw_bytes\nFROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence\nWHERE t.sequence>? AND t.sequence<=?\nAND m.role='assistant' AND m.content_bytes>0 ORDER BY t.sequence LIMIT ?",

			maxBytes, after, through, count)
	}
	// Existing message and unique source indexes avoid visiting every progress
	// event. Source bounds also prevent rescanning earlier conversation messages.
	return s.recordPage(ctx, "SELECT t.sequence,t.kind,CASE WHEN length(m.body)<=? THEN m.body END,m.refs,m.raw_bytes\nFROM messages m CROSS JOIN timeline t\nON t.kind='message' AND t.source_sequence=m.sequence\nWHERE m.sequence>CASE WHEN ?=0 THEN 0 ELSE\nCOALESCE((SELECT MAX(source_sequence) FROM timeline WHERE kind='message' AND sequence<=?),0) END\nAND m.sequence<=COALESCE((SELECT MAX(source_sequence) FROM timeline WHERE kind='message' AND sequence<=?),0)\nAND t.sequence>? AND t.sequence<=? AND m.role='assistant' AND m.content_bytes>0\nORDER BY t.sequence LIMIT ?",

		maxBytes, after, after, through, after, through, count)
}

func (s *Sessions) ResponseOutput(ctx context.Context, project, id string, after, through int64, maxBytes int) (text string, resultErr error) {
	store, err := s.open(ctx, project, id, true)
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, store.Close()) }()
	return store.ResponseOutput(ctx, id, after, through, maxBytes)
}
