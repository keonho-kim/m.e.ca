// Purpose: Open private session files with their public-room owner and receive position.
// Pattern: SQLite repository adapter.
// Usage: Team composition opens and leases a private Store before running a participant.
// Related: internal/storage/sqlite/team_registration.go, internal/commands/teamrequest/peer.go.
package sqlite

import (
	"context"
	"path/filepath"
)

func (s *Store) teamSessions() (*Sessions, error) {
	return OpenSessions(filepath.Join(filepath.Dir(s.path), ".team", s.session.ID))
}
func (s *Store) OpenTeamAgent(ctx context.Context, id string) (*Store, error) {
	sessions, err := s.teamSessions()
	if err != nil {
		return nil, err
	}
	child, err := sessions.Open(ctx, s.session.ProjectRoot, id)
	if err == nil {
		child.room = s
		s.teamContent.acquire()
		child.teamLease = &s.teamContent
	}
	return child, err
}
func (s *Store) ContextCursor(ctx context.Context, session string) (int64, error) {
	if err := s.checkSession(session); err != nil {
		return 0, err
	}
	var cursor int64
	err := s.db.QueryRowContext(ctx, "SELECT context_cursor FROM sessions WHERE id=?", session).Scan(&cursor)
	return cursor, err
}
