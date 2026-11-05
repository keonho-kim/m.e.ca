// Purpose: Project every active owned command and the latest completed passive status without private logs.
// Pattern: Bounded room-owned transient observation.
// Usage: Public commits update participant statuses; current-work context formats the immutable snapshot.
// Related: internal/agent/team/publication.go, internal/agent/team/work_context.go.
package team

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"unicode/utf8"
)

type commandStatus struct {
	from, text string
	status     string
	sequence   int64
}

func writeCommandStatuses(out *strings.Builder, statuses []commandStatus) {
	heading := false
	for _, status := range statuses {
		if status.sequence == 0 {
			continue
		}
		if !heading {
			out.WriteString("\nCurrent command observations:\n")
			heading = true
		}
		text := status.text
		fmt.Fprintf(out, "%s (#%d):\n%s\n", status.from, status.sequence, text)
	}
}

// Called at the same room-state boundary as the durable public cursor.
func (r *Room) updateCommandStatus(value channel.Message, process *toolapi.ProcessOutcome) {
	live := r.active[value.From]
	if live == nil {
		return
	}
	if process != nil {
		if toolapi.CommandActive(process.Status) {
			if live.statuses == nil {
				live.statuses = make(map[string]commandStatus)
			}
			live.statuses[process.CommandID] = commandStatus{from: value.From, text: compactCommandText(value.Text), status: process.Status, sequence: value.Sequence}
		} else {
			delete(live.statuses, process.CommandID)
			if len(live.statuses) == 0 {
				live.statuses = nil
			}
		}
		r.latestStatus = value.Sequence
	}
	if process != nil && toolapi.CommandActive(process.Status) {
		live.status = commandStatus{}
	} else if value.Passive {
		live.status = commandStatus{from: value.From, text: compactCommandText(value.Text), sequence: value.Sequence}
		r.latestStatus = value.Sequence
	} else if value.Observation && live.status.sequence != 0 {
		live.status = commandStatus{}
		r.latestStatus = value.Sequence
	}
}
func compactCommandText(text string) string {
	if len(text) <= 2048 {
		return text
	}
	end := 2048
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + "…"
}
