// Purpose: Publish freely addressed public feedback with authenticated references and no work assignment.
// Pattern: One-shot collaboration use case.
// Usage: The team-only discuss tool returns immediately after the ordered public append.
// Related: internal/agent/team/channel/attention.go, internal/tools/team/coordination.go.
package team

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

// note resolves only this room's committed records; model text cannot choose its sender or provenance.
func (p *Peer) note(ctx context.Context, replyTo int64, mentions []string) (*runrecord.TeamNote, error) {
	r := p.room
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if replyTo < 0 || len(mentions) > MaxActive {
		return nil, fmt.Errorf("invalid public discussion reference or mentions")
	}
	r.mu.Lock()
	through, based := r.latestPublic, p.DecisionCursor
	live := r.active[p.ID]
	valid := live != nil && !live.closed && !live.leaving
	var known [MaxActive]string
	count := 0
	for name := range r.active {
		known[count] = name
		count++
	}
	r.mu.Unlock()
	if !valid {
		return nil, fmt.Errorf("this team member is no longer active")
	}
	for i, name := range mentions {
		if slices.Contains(mentions[:i], name) {
			continue
		}
		if slices.Contains(known[:count], name) {
			continue
		}
		if _, err := r.options.Store.TeamMember(ctx, name); err != nil {
			return nil, fmt.Errorf("unknown teammate %q; use list_team_members: %w", name, err)
		}
	}
	note := &runrecord.TeamNote{ReplyTo: replyTo, Mentions: mentions, BasedOn: based}
	if replyTo > 0 {
		if replyTo > through {
			return nil, fmt.Errorf("public note #%d does not exist in this conversation", replyTo)
		}
		value, known := r.public.Metadata(replyTo)
		if !known {
			var err error
			value, err = r.options.Store.TeamMessageMetadata(ctx, replyTo)
			if err != nil {
				return nil, err
			}
		}
		if value.Sequence != replyTo {
			return nil, fmt.Errorf("public note #%d does not exist in this conversation", replyTo)
		}
		note.ReplyAuthor = value.From
	}
	return note, nil
}

func (p *Peer) Discuss(ctx context.Context, text string, replyTo int64, mentions []string) (sequence int64, resultErr error) {
	if strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("public feedback must contain a useful message")
	}
	if err := p.room.admitUser(); err != nil {
		return 0, err
	}
	note, err := p.note(ctx, replyTo, mentions)
	if err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r := p.room
	r.logMu.Lock()
	defer func() { resultErr = errors.Join(resultErr, r.endPublication()) }()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r.mu.Lock()
	err = p.checkUserDecisionLocked()
	r.mu.Unlock()
	if err != nil {
		return 0, err
	}
	value, err := r.appendPublic(channel.Message{From: p.ID, Text: text, Discussion: true, Note: note})
	return value.Sequence, err
}
