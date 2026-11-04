// Purpose: Render compact current work and preserve its complete public-source references.
// Pattern: Immutable snapshot and pure bounded presentation.
// Usage: Room snapshots state under its mutex, then formats outside the shared lock.
// Related: internal/agent/team/contribution/state.go, internal/agent/team/work_context.go.
package contribution

import (
	"fmt"
	"slices"
	"strings"
)

// These repeated-context budgets do not restrict tool inputs or stored declarations.
const pathContextBytes = 24 * 1024

// Snapshot shares frozen path arrays; replacing scopes never mutates an older snapshot.
func (s *State) Snapshot() []Scope {
	out := make([]Scope, 0, len(s.scopes))
	for _, value := range s.scopes {
		out = append(out, value)
	}
	slices.SortFunc(out, func(a, b Scope) int { return strings.Compare(a.Member, b.Member) })
	return out
}

// WriteContext retains the current reference when its complete declaration is already visible.
func WriteContext(out *strings.Builder, value Scope, covered bool) {
	out.WriteString(value.Member)
	out.WriteString(": ")
	if covered {
		fmt.Fprintf(out, "current work #%d, revision %d (details in conversation)\n", value.Sequence, value.Revision)
		return
	}
	out.WriteString(value.Summary)
	out.WriteByte('\n')
	if value.Sequence > 0 {
		fmt.Fprintf(out, "  Work #%d, revision %d\n", value.Sequence, value.Revision)
	}
	if value.Contract != "" {
		out.WriteString("  Declared interface/behavior: ")
		out.WriteString(value.Contract)
		out.WriteByte('\n')
	}
	if value.ReplyTo > 0 {
		fmt.Fprintf(out, "  Refers to public note #%d\n", value.ReplyTo)
	}
	if len(value.Paths) == 0 {
		out.WriteString("  No paths reserved; this does not establish completion.\n")
	}
	bytes := 0
	for i, path := range value.Paths {
		if bytes+len(path)+3 > pathContextBytes {
			fmt.Fprintf(out, "  %d more paths; list_team_members has the full scope.\n", len(value.Paths)-i)
			break
		}
		out.WriteString("  ")
		out.WriteString(path)
		out.WriteByte('\n')
		bytes += len(path) + 3
	}
}
