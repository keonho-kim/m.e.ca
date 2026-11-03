// Purpose: Register a freshly created private session and remove known unregistered files on failure.
// Pattern: Repository-owned creation transaction with exact compensating cleanup.
// Usage: Team invitations use CreateTeamAgent as the only identity publication boundary.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/team_members.go.
package sqlite

import (
	"context"
	"errors"
	teamidentity "github.com/keonho-kim/meca/internal/agent/team/identity"
	"os"
	"time"
)

func (s *Store) CreateTeamAgent(ctx context.Context) (member teamidentity.Member, resultErr error) {
	sessions, err := s.teamSessions()
	if err != nil {
		return member, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return member, err
	}
	defer tx.Rollback()
	var ordinal int
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(ordinal),-1)+1 FROM team_members").Scan(&ordinal); err != nil {
		return member, err
	}
	child, err := sessions.Create(ctx, s.session.ProjectRoot, s.session.DisplayRoot)
	if err != nil {
		return member, err
	}
	member = teamidentity.Member{ID: teamidentity.Name(ordinal), Session: child.session.ID}
	path := child.path
	defer func() {
		if resultErr == nil {
			return
		}
		// A cancelled/uncertain commit is checked independently before deleting only
		// the exact new empty database. Existing/restored sessions are never removed.
		_ = tx.Rollback()
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		var registered bool
		check := s.db.QueryRowContext(flush, "SELECT EXISTS(SELECT 1 FROM team_members WHERE session_id=?)", member.Session).Scan(&registered)
		if check != nil {
			resultErr = errors.Join(resultErr, check)
			return
		}
		if registered {
			return
		}
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if err := os.Remove(path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
				resultErr = errors.Join(resultErr, err)
			}
		}
	}()
	if err := child.Close(); err != nil {
		return member, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO team_members(ordinal,agent_id,session_id) VALUES(?,?,?)", ordinal, member.ID, member.Session); err != nil {
		return member, err
	}
	return member, tx.Commit()
}
