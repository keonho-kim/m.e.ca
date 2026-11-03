// Purpose: Select compact terminal history when the repository supports on-demand tool details.
// Pattern: Consumer-owned optional read capability.
// Usage: Initial restoration and older-page workers use the same history source.
// Related: internal/storage/sqlite/timeline.go, internal/tui/session/history.go.
package session

import (
	"context"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func terminalHistory(ctx context.Context, store Store, id string, before int64, limit int) ([]history.Record, int64, error) {
	if compact, ok := store.(interface {
		TimelineSummaries(context.Context, string, int64, int) ([]history.Record, int64, error)
	}); ok {
		return compact.TimelineSummaries(ctx, id, before, limit)
	}
	return store.Timeline(ctx, id, before, limit)
}
