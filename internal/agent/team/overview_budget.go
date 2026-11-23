// Purpose: Keep repeated team summaries within their UTF-8 byte budget.
// Pattern: Bounded presentation buffer with an explicit history lookup hint.
// Usage: Team context projections share bounded cached summary sections.
// Related: internal/agent/team/work_context.go, internal/agent/team/context_projection.go, internal/agent/team/answer.go.
package team

import (
	"strings"
	"unicode/utf8"
)

const (
	teamOverviewBytes  = 256 * 1024
	teamOverviewTitle  = "Current team work (observed facts):\n"
	teamOverviewNotice = "\nTeam overview shortened. Use list_team_members for full work scopes and read_team_messages for public details.\n"
	teamIdleGuidance   = "All active teammates have finished responding and no command is running. If responsibilities or the Spec are unresolved, use discuss to propose or accept the missing agreement before implementation. Otherwise act on a ready contribution within the agreed scope, or use leave_team if your contribution is complete. State a concrete blocker rather than repeating a general question about what to do."
	overviewBodyBytes  = teamOverviewBytes - len(teamOverviewNotice) - len(teamIdleGuidance) - 1
)

type overviewPart struct {
	text      string
	shortened bool
}

type overviewBuffer struct {
	body      strings.Builder
	shortened bool
}

// Writers can finish their current row after clipping without retaining its tail.
func (b *overviewBuffer) WriteString(text string) (int, error) {
	n := len(text)
	if b.shortened {
		return n, nil
	}
	if limit := overviewBodyBytes - b.body.Len(); n > limit {
		for limit > 0 && !utf8.RuneStart(text[limit]) {
			limit--
		}
		text = text[:limit]
		b.shortened = true
	}
	b.body.WriteString(text)
	return n, nil
}

func (b *overviewBuffer) Write(value []byte) (int, error) {
	return b.WriteString(string(value))
}

func (b *overviewBuffer) WriteByte(value byte) error {
	if b.body.Len() == overviewBodyBytes {
		b.shortened = true
	}
	if !b.shortened {
		b.body.WriteByte(value)
	}
	return nil
}

func (b *overviewBuffer) add(part overviewPart) {
	b.WriteString(part.text)
	b.shortened = b.shortened || part.shortened
}

func (b *overviewBuffer) part() overviewPart {
	return overviewPart{text: b.body.String(), shortened: b.shortened}
}

func (b *overviewBuffer) finish() string {
	if b.shortened {
		b.body.WriteString(teamOverviewNotice)
	}
	return b.body.String()
}
