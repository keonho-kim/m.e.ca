// Purpose: Persist the selected conversation's team-mode setting.
// Pattern: SQLite repository adapter.
// Usage: The TUI and headless composition read the mode independently of active participants.
// Related: internal/tui/session/team_state.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
)

func (s *Store) TeamMode(ctx context.Context) (bool, error) {
	var enabled bool
	err := s.db.QueryRowContext(ctx, "SELECT team_enabled FROM sessions WHERE singleton=1").Scan(&enabled)
	return enabled, err
}
func (s *Store) SetTeamMode(ctx context.Context, enabled bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE sessions SET team_enabled=? WHERE singleton=1", enabled)
	return err
}
