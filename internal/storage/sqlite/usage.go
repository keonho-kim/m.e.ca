// Purpose: Restore reported token counts and validate persisted usage.
// Pattern: Repository projection.
// Usage: Session restoration reads TokenUsage once before starting a worker.
// Related: internal/storage/sqlite/events.go, internal/agent/run/usage.go.
package sqlite

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *Store) TokenUsage(ctx context.Context, session string) (modelapi.TokenUsage, error) {
	var total modelapi.TokenUsage
	var invalid int
	err := s.db.QueryRowContext(ctx, `SELECT
	 COALESCE(SUM(json_extract(body,'$.usage.input')),0),
	 COALESCE(SUM(json_extract(body,'$.usage.output')),0),
	 COALESCE(SUM(CASE WHEN
	  json_type(body,'$.usage.input') IS NOT 'integer' OR
	  json_type(body,'$.usage.output') IS NOT 'integer' OR
	  EXISTS (SELECT 1 FROM json_each(body,'$.usage') WHERE key NOT IN ('input','output')) OR
	  json_extract(body,'$.usage.input') NOT BETWEEN 0 AND ? OR
	  json_extract(body,'$.usage.output') NOT BETWEEN 0 AND ?
	 THEN 1 ELSE 0 END),0)
	 FROM tool_events WHERE session_id=? AND json_extract(body,'$.kind')='usage'`, modelapi.MaxCallTokens, modelapi.MaxCallTokens, session).Scan(&total.Input, &total.Output, &invalid)
	if err != nil {
		return total, err
	}
	if invalid != 0 {
		return modelapi.TokenUsage{}, fmt.Errorf("invalid stored token usage: expected only provider-reported input/output counts; records were not modified")
	}
	return total, nil
}
