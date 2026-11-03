// Purpose: Own persistent peer identities and stable human-readable names without starting execution.
// Pattern: Domain values and pure identity parsing.
// Usage: Membership, persistence and team request composition share these values.
// Related: internal/agent/team/types.go, internal/storage/sqlite/team_members.go.
package identity

import "strings"

type Member struct {
	ID      string `json:"id"`
	Session string `json:"session"`
	Summary string `json:"summary,omitempty"`
}

func ValidName(id string) bool {
	if !strings.HasPrefix(id, "agent-") || len(id) < 7 || len(id) > 40 {
		return false
	}
	for _, c := range id[6:] {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}
func Name(index int) string {
	value := ""
	for {
		value = string(rune('a'+index%26)) + value
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	return "agent-" + value
}
