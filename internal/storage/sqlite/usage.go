// Purpose: Restore verified token totals and increment them from new durable usage records.
// Pattern: Incremental projection over the append-only session journal.
// Usage: TUI restoration and native sessionUsage share identity-scoped aggregates.
// Related: internal/storage/sqlite/usage_cache.go, internal/storage/sqlite/events.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// TokenUsage reads only the session ledger, without model or tool runtime restoration.
func (s *Sessions) TokenUsage(ctx context.Context, project, session string) (total modelapi.TokenUsage, resultErr error) {
	store, err := s.OpenRead(ctx, project, session)
	if err != nil {
		return total, err
	}
	defer func() { resultErr = errors.Join(resultErr, store.Close()) }()
	return store.TokenUsage(ctx, session)
}

func (s *Store) TokenUsage(ctx context.Context, session string) (modelapi.TokenUsage, error) {
	if err := s.checkSession(session); err != nil {
		return modelapi.TokenUsage{}, err
	}
	projection := s.usage
	if projection == nil {
		projection = &s.localUsage
	}
	projection.mu.Lock()
	defer projection.mu.Unlock()
	var delta modelapi.TokenUsage
	var cost sql.NullFloat64
	var invalid int
	var sequence int64
	err := s.db.QueryRowContext(ctx, `SELECT
	 COALESCE(MAX(sequence),?),
	 COALESCE(SUM(json_extract(body,'$.usage.input')),0),
	 COALESCE(SUM(json_extract(body,'$.usage.output')),0),
	 SUM(json_extract(body,'$.usage.cost')),
	 COALESCE(SUM(CASE WHEN
	  json_type(body,'$.usage.input') IS NOT 'integer' OR
	  json_type(body,'$.usage.output') IS NOT 'integer' OR
	  EXISTS (SELECT 1 FROM json_each(body,'$.usage') WHERE key NOT IN ('input','output','cost')) OR
	  (json_type(body,'$.usage.cost') IS NOT NULL AND json_type(body,'$.usage.cost') NOT IN ('integer','real','null')) OR
	  json_extract(body,'$.usage.cost') < 0 OR
	  json_extract(body,'$.usage.input') NOT BETWEEN 0 AND ? OR
	  json_extract(body,'$.usage.output') NOT BETWEEN 0 AND ?
	 THEN 1 ELSE 0 END),0)
	 FROM tool_events WHERE kind='usage' AND sequence>?`,
		projection.sequence, modelapi.MaxCallTokens, modelapi.MaxCallTokens, projection.sequence).
		Scan(&sequence, &delta.Input, &delta.Output, &cost, &invalid)
	if err != nil {
		return modelapi.TokenUsage{}, err
	}
	if invalid != 0 {
		return modelapi.TokenUsage{}, faults.Errorf(diagcatalog.SqliteInvalidStoredUsageExpectedProviderReportedCounts)
	}
	if sequence == projection.sequence {
		return projection.total.Clone(), nil
	}
	if cost.Valid {
		delta.Cost = &cost.Float64
	}
	total := projection.total.Add(delta)
	if err := total.CheckTotals(); err != nil {
		return modelapi.TokenUsage{}, err
	}
	// A failed tail never advances the cursor or hides invalid durable records.
	projection.sequence, projection.total = sequence, total
	return total.Clone(), nil
}
