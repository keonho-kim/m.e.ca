// Purpose: Publish freely addressed public feedback with authenticated references and no work assignment.
// Pattern: One-shot collaboration use case.
// Usage: The team-only discuss tool returns immediately after the ordered public append.
// Related: internal/agent/team/channel/attention.go, internal/tools/team/coordination.go.
package team

import (
	"context"
	"errors"

	"slices"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// note resolves only this room's committed records; model text cannot choose its sender or provenance.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (p *Peer) note(ctx context.Context, replyTo int64, mentions []string) (*runrecord.TeamNote, error) {
	r := p.room
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if replyTo < 0 || len(mentions) > MaxMentions {
		return nil, faults.Errorf(diagcatalog.TeamInvalidPublicDiscussionReferenceOrMentions)
	}
	r.mu.Lock()
	through, based := r.latestPublic, p.DecisionCursor
	live := r.active[p.ID]
	valid := live != nil && !live.closed && !live.leaving
	var known [MaxMentions]bool
	for i, name := range mentions {
		known[i] = r.active[name] != nil
	}
	r.mu.Unlock()
	if !valid {
		return nil, faults.Errorf(diagcatalog.TeamThisTeamMemberIsNoLongerActive)
	}
	for i, name := range mentions {
		if name == p.ID {
			return nil, faults.Errorf(diagcatalog.TeamIsYourOwnNameNotAnotherTeammate, name)
		}
		if slices.Contains(mentions[:i], name) {
			continue
		}
		if known[i] {
			continue
		}
		if _, err := r.options.Store.TeamMember(ctx, name); err != nil {
			return nil, faults.Errorf(diagcatalog.TeamUnknownTeammateUseListTeamMembers, name, err)
		}
	}
	note := &runrecord.TeamNote{ReplyTo: replyTo, Mentions: mentions, BasedOn: based}
	if replyTo > 0 {
		value, err := r.publicMetadata(ctx, through, replyTo)
		if err != nil {
			return nil, err
		}
		note.ReplyAuthor = value.From
	}
	return note, nil
}

func (r *Room) publicMetadata(ctx context.Context, through, sequence int64) (channel.Metadata, error) {
	if sequence > through {
		return channel.Metadata{}, faults.Errorf(diagcatalog.TeamPublicNoteDoesNotExistInThis, sequence)
	}
	value, known := r.public.Metadata(sequence)
	if !known {
		var err error
		value, err = r.options.Store.TeamMessageMetadata(ctx, sequence)
		if err != nil {
			return channel.Metadata{}, err
		}
	}
	if value.Sequence != sequence {
		return channel.Metadata{}, faults.Errorf(diagcatalog.TeamPublicNoteDoesNotExistInThis, sequence)
	}
	return value, nil
}

func (r *Room) workReference(ctx context.Context, sequence int64) (channel.Metadata, error) {
	if sequence == 0 {
		return channel.Metadata{}, nil
	}
	r.mu.Lock()
	through := r.latestPublic
	r.mu.Unlock()
	value, err := r.publicMetadata(ctx, through, sequence)
	if err != nil {
		return channel.Metadata{}, err
	}
	if value.Kind != "team_discussion" {
		return channel.Metadata{}, faults.Errorf(diagcatalog.TeamWorkReferenceMustIdentifyPublicProposal)
	}
	return value, nil
}

func (p *Peer) Discuss(ctx context.Context, text string, replyTo int64, mentions []string) (sequence int64, resultErr error) {
	if strings.TrimSpace(text) == "" {
		return 0, faults.Errorf(diagcatalog.TeamPublicFeedbackMustContainAUsefulMessage)
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
