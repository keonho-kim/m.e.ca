// Purpose: Admit user steering and assemble unread public observations at safe model boundaries.
// Pattern: Bounded receive-cursor projection.
// Usage: The team runner calls Inbox before decisions, tools and post-compaction continuation.
// Related: internal/agent/team/publication.go, internal/agent/run/context_inbox.go.
package team

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func (r *Room) admitUser() error {
	if r.options.PollSteering == nil {
		return nil
	}
	r.inputMu.Lock()
	defer r.inputMu.Unlock()
	r.mu.Lock()
	finished := r.finished
	r.mu.Unlock()
	if finished {
		return nil
	}
	for _, text := range r.options.PollSteering() {
		value, err := r.publish(channel.Message{From: "user", Text: text})
		if err != nil {
			return err
		}
		r.mu.Lock()
		r.input = value.Text
		r.mu.Unlock()
	}
	return nil
}
func (p *Peer) Inbox(ctx context.Context) (run.ContextBatch, error) {
	r := p.room
	if err := r.admitUser(); err != nil {
		return run.ContextBatch{}, err
	}
	view := r.workSnapshot()
	through := view.through
	batch := run.ContextBatch{}
	bytes := 0
receive:
	for p.Cursor < through {
		messages, err := r.public.Read(ctx, r.options.Store, p.Cursor, through, 100)
		if err != nil {
			return batch, err
		}
		if len(messages) == 0 {
			break
		}
		for _, observation := range messages {
			message := observation.Message
			if !message.Passive && (message.From != p.ID || message.OwnContext()) {
				value := observation.Context
				if bytes > 0 && bytes+len(value.Content) > channel.PageBytes {
					break receive
				}
				batch.Messages = append(batch.Messages, value)
				bytes += len(value.Content)
				if message.From == "user" {
					batch.Objective = message.Text
					p.UserSequence = message.Sequence
				}
			}
			p.Cursor = message.Sequence
			batch.Cursor = message.Sequence
		}
		if bytes >= channel.PageBytes {
			break
		}
	}
	r.mu.Lock()
	if p.Cursor >= through {
		p.receivedView = view
	} else {
		// A bounded receive page cannot use a newer snapshot as if its intervening notes were seen.
		p.receivedView = decisionSnapshot{through: p.Cursor, view: &workProjection{text: fmt.Sprintf("Public notes received through #%d. More recent notes remain available with read_team_messages; list_team_members provides current contributions.", p.Cursor)}}
		batch.Reconsider = true
	}
	if live := r.active[p.ID]; live != nil {
		batch.Reconsider = batch.Reconsider || live.attention > p.DecisionCursor && live.attention <= p.Cursor
	}
	if r.latestUser <= p.Cursor {
		p.UserSequence = max(p.UserSequence, r.latestUser)
	}
	batch.UserSequence = p.UserSequence
	r.mu.Unlock()
	return batch, nil
}
func (p *Peer) ReadMessages(ctx context.Context, after int64) ([]channel.Message, error) {
	r := p.room
	r.mu.Lock()
	through := r.latestPublic
	r.mu.Unlock()
	values, err := r.public.Read(ctx, r.options.Store, after, through, 100)
	if err != nil {
		return nil, err
	}
	messages := make([]channel.Message, len(values))
	for i, value := range values {
		messages[i] = value.Message
	}
	return messages, nil
}
