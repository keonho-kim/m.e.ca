// Purpose: Commit and announce ordered public messages without holding room state during I/O.
// Pattern: Serialized append boundary with short state updates.
// Usage: User input, progress and contribution transitions publish through Room.
// Related: internal/agent/team/inbox.go, internal/agent/team/events.go.
package team

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"strings"
)

func (r *Room) publish(message channel.Message) (channel.Message, error) {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	return r.appendPublic(message)
}

// appendPublic requires logMu, never mu: SQLite and slow UI consumers must not
// prevent another peer from inspecting membership or admitting existing context.
func (r *Room) appendPublic(message channel.Message) (channel.Message, error) {
	if r.options.Redact != nil {
		message.Text = r.options.Redact(message.Text)
	}
	value, err := r.options.Store.AppendTeamMessage(r.ctx, message)
	if err != nil {
		r.fail(err)
		return value, err
	}
	r.mu.Lock()
	previous := r.latestPublic
	r.mu.Unlock()
	r.public.append(value, previous)
	r.mu.Lock()
	r.latestPublic = value.Sequence
	if value.From == "user" {
		r.latestUser = value.Sequence
	}
	if live := r.active[value.From]; live != nil && value.FromStream && strings.TrimSpace(value.Text) != "" {
		live.spoke = true
	}
	for id, live := range r.active {
		if id != value.From && !live.closed && !live.leaving {
			wake(live)
		}
	}
	r.mu.Unlock()
	event := runrecord.Event{Kind: "team_message", Agent: value.From, Final: value.Final, Text: value.Text}
	if value.FromStream {
		event.Kind = "team_progress"
	}
	if value.From == "user" {
		event.Kind = "user_message"
	}
	return value, r.deliver(event, false)
}
func wake(live *participant) {
	select {
	case live.wake <- struct{}{}:
	default:
	}
}
