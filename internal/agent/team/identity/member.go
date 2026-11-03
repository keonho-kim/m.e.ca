// Purpose: Own persistent peer identities and stable human-readable names without starting execution.
// Pattern: Domain values and pure identity parsing.
// Usage: Membership, persistence and team request composition share these values.
// Related: internal/agent/team/types.go, internal/storage/sqlite/team_members.go.
package identity

import "slices"

type Member struct {
	ID      string `json:"id"`
	Session string `json:"session"`
	Summary string `json:"summary,omitempty"`
}

func ValidName(id string) bool { return slices.Contains(names[:], id) }
