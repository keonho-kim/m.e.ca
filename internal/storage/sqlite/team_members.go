// Purpose: Page and update stable team identities and contribution summaries.
// Pattern: SQLite repository adapter.
// Usage: The team runtime reads only the initial pair, a requested identity or a bounded page.
// Related: internal/storage/sqlite/team_registration.go, internal/agent/team/identity/member.go.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	teamidentity "github.com/keonho-kim/meca/internal/agent/team/identity"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Store) TeamMembers(ctx context.Context, after string, limit int) ([]teamidentity.Member, error) {
	if limit < 1 || limit > 200 || after != "" && !teamidentity.ValidName(after) {
		return nil, fmt.Errorf("invalid peer page")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT agent_id,session_id,summary FROM team_members
 WHERE ordinal>COALESCE((SELECT ordinal FROM team_members WHERE agent_id=?),-1) ORDER BY ordinal LIMIT ?`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := make([]teamidentity.Member, 0, limit)
	for rows.Next() {
		member, err := readMember(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}
func (s *Store) TeamMember(ctx context.Context, id string) (teamidentity.Member, error) {
	if !teamidentity.ValidName(id) {
		return teamidentity.Member{}, fmt.Errorf("invalid peer ID")
	}
	member, err := readMember(s.db.QueryRowContext(ctx, "SELECT agent_id,session_id,summary FROM team_members WHERE agent_id=?", id))
	if err == sql.ErrNoRows {
		return member, fmt.Errorf("unknown peer %q; use list_friends", id)
	}
	return member, err
}
func readMember(row interface{ Scan(...any) error }) (teamidentity.Member, error) {
	var member teamidentity.Member
	if err := row.Scan(&member.ID, &member.Session, &member.Summary); err != nil {
		return member, err
	}
	if !teamidentity.ValidName(member.ID) || !sessionstate.ValidID(member.Session) {
		return member, fmt.Errorf("invalid stored team identity")
	}
	return member, nil
}
func (s *Store) SetTeamSummary(ctx context.Context, id, summary string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE team_members SET summary=? WHERE agent_id=?", summary, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n != 1 {
		err = fmt.Errorf("unknown peer %q", id)
	}
	return err
}
