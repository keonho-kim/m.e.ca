// Purpose: Project compact current contributions while preserving complete declarations with their owner.
// Pattern: Immutable snapshot and pure bounded presentation.
// Usage: Room snapshots state under its mutex, then formats outside the shared lock.
// Related: internal/agent/team/contribution/state.go, internal/agent/team/work_context.go.
package contribution

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// These repeated-context budgets do not restrict tool inputs or stored declarations.
const summaryContextBytes = 4 * 1024
const pathContextBytes = 24 * 1024

func summaryPreview(summary string) string {
	if len(summary) <= summaryContextBytes {
		return summary
	}
	summary = summary[:summaryContextBytes]
	for !utf8.ValidString(summary) {
		summary = summary[:len(summary)-1]
	}
	return summary + "… (full declaration in public conversation)"
}

// Overview preserves all owned paths while bounding only the explanatory summary.
func (s *State) Overview(member string) Scope {
	value := s.Scope(member)
	value.Summary = summaryPreview(value.Summary)
	return value
}

// Snapshot owns its slice headers so formatting can run outside the room mutex.
func (s *State) Snapshot() []Scope {
	out := make([]Scope, 0, len(s.scopes))
	for _, value := range s.scopes {
		value.Paths = slices.Clone(value.Paths)
		out = append(out, value)
	}
	slices.SortFunc(out, func(a, b Scope) int { return strings.Compare(a.Member, b.Member) })
	return out
}

func Context(scopes []Scope) string {
	var out strings.Builder
	for _, value := range scopes {
		out.WriteString(value.Member)
		out.WriteString(": ")
		out.WriteString(summaryPreview(value.Summary))
		out.WriteByte('\n')
		bytes := 0
		for i, path := range value.Paths {
			if bytes+len(path)+3 > pathContextBytes {
				fmt.Fprintf(&out, "  %d more paths; list_team_members has the full scope.\n", len(value.Paths)-i)
				break
			}
			out.WriteString("  ")
			out.WriteString(path)
			out.WriteByte('\n')
			bytes += len(path) + 3
		}
	}
	return out.String()
}
