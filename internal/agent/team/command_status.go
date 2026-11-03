// Purpose: Project one latest passive command status per active member without accumulating private history.
// Pattern: Bounded room-owned transient observation.
// Usage: Public commits update participant.status; current-work context formats the immutable snapshot.
// Related: internal/agent/team/publication.go, internal/agent/team/work_context.go.
package team

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type commandStatus struct {
	from, text string
	sequence   int64
}

func writeCommandStatuses(out *strings.Builder, statuses []commandStatus) {
	heading := false
	for _, status := range statuses {
		if status.sequence == 0 {
			continue
		}
		if !heading {
			out.WriteString("\nLatest command observations:\n")
			heading = true
		}
		text := status.text
		if len(text) > 2*1024 {
			text = text[:2*1024]
			for !utf8.ValidString(text) {
				text = text[:len(text)-1]
			}
			text += "…"
		}
		fmt.Fprintf(out, "%s (#%d):\n%s\n", status.from, status.sequence, text)
	}
}
