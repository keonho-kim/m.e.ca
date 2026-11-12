// Purpose: Register a freshly created private session and remove known unregistered files on failure.
// Pattern: Repository-owned creation transaction with exact compensating cleanup.
// Usage: Team invitations use CreateTeamAgent as the only identity publication boundary.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/team_members.go.
package sqlite

import (
	"context"
	"errors"

	"os"
	"time"

	teamidentity "github.com/keonho-kim/meca/internal/agent/team/identity"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) CreateTeamAgent(ctx context.Context) (member teamidentity.Member, resultErr error) {
	sessions, err := s.teamSessions()
	if err != nil {
		return member, err
	}
	child, err := sessions.Create(ctx, s.session.ProjectRoot, s.session.DisplayRoot)
	if err != nil {
		return member, err
	}
	member.Session = child.session.ID
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, s.removeUnregisteredTeamSession(ctx, member.Session, child.path))
		}
	}()
	if err := child.Close(); err != nil {
		return member, err
	}
	// Filesystem preparation never holds the public journal's sole DB connection.
	tx, releaseWrite, err := s.beginWrite(ctx)
	if err != nil {
		return member, err
	}
	defer releaseWrite()
	defer tx.Rollback()
	var ordinal int
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(ordinal),-1)+1 FROM team_members").Scan(&ordinal); err != nil {
		return member, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT agent_id FROM team_members")
	if err != nil {
		return member, err
	}
	var used []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			break
		}
		used = append(used, name)
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return member, err
	}
	name, err := teamidentity.Choose(used)
	if err != nil {
		return member, err
	}

	member.ID = name
	if _, err := tx.ExecContext(ctx, "INSERT INTO team_members(ordinal,agent_id,session_id) VALUES(?,?,?)", ordinal, member.ID, member.Session); err != nil {
		return member, err
	}
	return member, tx.Commit()
}

// DiscardTeamAgent removes only a freshly registered, closed session that the
// room has not admitted or published. Recorded work is never a cleanup target.
func (s *Store) DiscardTeamAgent(ctx context.Context, member teamidentity.Member) error {
	sessions, err := s.teamSessions()
	if err != nil {
		return err
	}
	child, err := sessions.Open(ctx, s.session.ProjectRoot, member.Session)
	if err != nil {
		return err
	}
	var used bool
	err = child.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM timeline) OR EXISTS(SELECT 1 FROM run_states) OR EXISTS(SELECT 1 FROM team_members)").Scan(&used)
	err = errors.Join(err, child.Close())
	if err != nil {
		return err
	}
	if used {
		return faults.Errorf(diagcatalog.SqliteCannotDiscardATeamSessionContainingRecorded)
	}
	result, err := s.execWrite(ctx, "DELETE FROM team_members WHERE agent_id=? AND session_id=? AND summary=''", member.ID, member.Session)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return faults.Errorf(diagcatalog.SqliteUnstartedTeamRegistrationDoesNotMatch)
	}
	return s.removeUnregisteredTeamSession(ctx, member.Session, child.path)
}

// removeUnregisteredTeamSession checks uncertain commits before deleting only the new DB.
func (s *Store) removeUnregisteredTeamSession(ctx context.Context, session, path string) error {
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var registered bool
	if err := s.db.QueryRowContext(flush, "SELECT EXISTS(SELECT 1 FROM team_members WHERE session_id=?)", session).Scan(&registered); err != nil {
		return err
	}
	if registered {
		return nil
	}
	var result error
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	return result
}
