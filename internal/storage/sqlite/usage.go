// Purpose: Restore provider-reported token and charge totals from persisted usage events.
// Pattern: Repository projection.
// Usage: Session restoration reads TokenUsage once before starting a worker.
// Related: internal/storage/sqlite/events.go, internal/agent/run/usage.go.
package sqlite

import (
	"context"
	"database/sql"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) TokenUsage(ctx context.Context, session string) (modelapi.TokenUsage, error) {
	if err := s.checkSession(session); err != nil {
		return modelapi.TokenUsage{}, err
	}
	var total modelapi.TokenUsage
	var cost sql.NullFloat64
	var invalid int
	err := s.db.QueryRowContext(ctx, "SELECT\n\t COALESCE(SUM(json_extract(body,'$.usage.input')),0),\n\t COALESCE(SUM(json_extract(body,'$.usage.output')),0),\n\t SUM(json_extract(body,'$.usage.cost')),\n\t COALESCE(SUM(CASE WHEN\n\t  json_type(body,'$.usage.input') IS NOT 'integer' OR\n\t  json_type(body,'$.usage.output') IS NOT 'integer' OR\n\t  EXISTS (SELECT 1 FROM json_each(body,'$.usage') WHERE key NOT IN ('input','output','cost')) OR\n\t  (json_type(body,'$.usage.cost') IS NOT NULL AND json_type(body,'$.usage.cost') NOT IN ('integer','real','null')) OR\n\t  json_extract(body,'$.usage.cost') < 0 OR\n\t  json_extract(body,'$.usage.input') NOT BETWEEN 0 AND ? OR\n\t  json_extract(body,'$.usage.output') NOT BETWEEN 0 AND ?\n\t THEN 1 ELSE 0 END),0)\n\t FROM tool_events WHERE kind='usage'",

		modelapi.MaxCallTokens, modelapi.MaxCallTokens).Scan(&total.Input, &total.Output, &cost, &invalid)
	if err != nil {
		return total, err
	}
	if invalid != 0 {
		return modelapi.TokenUsage{}, faults.Errorf(diagcatalog.SqliteInvalidStoredUsageExpectedProviderReportedCounts)
	}
	if cost.Valid {
		total.Cost = &cost.Float64
	}
	if err := total.CheckTotals(); err != nil {
		return modelapi.TokenUsage{}, err
	}
	return total, nil
}
