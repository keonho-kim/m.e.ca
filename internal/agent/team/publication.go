// Purpose: Commit and announce ordered public messages without holding room state during I/O.
// Pattern: Serialized append boundary with short state updates.
// Usage: User input, progress and contribution transitions publish through Room.
// Related: internal/agent/team/inbox.go, internal/agent/team/events.go.
package team

import (
	"context"
	"errors"
	"strings"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

// publicEffect carries transient state transitions, never additional persisted metadata.
type publicEffect struct {
	command  *toolapi.ProcessOutcome
	work     *contribution.Scope
	evidence *[32]byte
}

func (r *Room) publish(message channel.Message) (result channel.Message, resultErr error) {
	r.logMu.Lock()
	defer func() { resultErr = errors.Join(resultErr, r.endPublication()) }()
	return r.appendPublic(message)
}

// appendPublic requires logMu, never mu. Presentation is drained after this
// ordered durable boundary has been released.
func (r *Room) appendPublic(message channel.Message) (channel.Message, error) {
	value, err := r.storePublic(message)
	if err != nil {
		return value, err
	}
	return r.acceptPublic(value, publicEffect{})
}

// storePublic requires logMu and freezes caller-owned metadata before durable publication.
func (r *Room) storePublic(message channel.Message) (channel.Message, error) {
	message.Note = message.Note.Clone()
	if r.options.Redact != nil {
		message.Text = r.options.Redact(message.Text)
		if message.Note != nil && message.Note.Work != nil {
			message.Note.Work.Contract = r.options.Redact(message.Note.Work.Contract)
		}
	}
	ctx := r.ctx
	if message.Reference != nil {
		// An already committed private result must remain reachable after Esc or a deadline.
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		ctx = flush
	}
	value, err := r.options.Store.AppendTeamMessage(ctx, message)
	if err != nil {
		r.fail(err)
		return value, err
	}
	return value, nil
}

// acceptPublic makes the log cursor and related work visible at the same state boundary.
func (r *Room) acceptPublic(value channel.Message, effect publicEffect) (channel.Message, error) {
	work := effect.work
	var discussion discussionPreview
	if value.Discussion && value.Note != nil {
		discussion = previewDiscussion(value)
	}
	r.mu.Lock()
	previous := r.latestPublic
	r.mu.Unlock()
	fact := value
	fact.ChangedPaths, fact.Reference = nil, nil
	r.public.Append(fact, previous)
	r.mu.Lock()
	notifyEvidence := true
	if effect.evidence != nil {
		fresh := r.recordProgress(*effect.evidence)
		// Actual or uncertain file effects still require fresh inspection, even
		// when an earlier operation produced the same resulting fingerprint.
		notifyEvidence = fresh || len(value.ChangedPaths) > 0
	}
	var notify [MaxActive]*participant
	n := 0
	for id, live := range r.active {
		if notifyEvidence && !live.closed && !live.leaving && r.needsAttentionLocked(value, id, work) {
			notify[n] = live
			n++
		}
	}
	if work != nil {
		if !r.work.SameCoordination(*work) && (len(work.Paths) > 0 || r.work.ContractChanged(*work) || !r.work.HasPaths(work.Member, nil)) {
			r.recordProgress(work.ProgressKey())
		}
		work.Paths = value.Note.Work.Paths
		r.work.Commit(*work, value.Sequence)
		r.workView = nil
		if live := r.active[value.From]; live != nil {
			live.spoke = true
		}
	}
	if discussion.sequence > 0 {
		if live := r.active[value.From]; live != nil {
			live.discussion = discussion
			r.latestDiscussion = value.Sequence
		}
	}
	r.updateCommandStatus(value, effect.command)

	if value.Observation {
		r.work.Record(value.From, value.ChangedPaths, value.Sequence)
	}
	if value.From == "user" {
		r.progress++
	}
	r.latestPublic = value.Sequence
	if value.From == "user" {
		r.latestUser = value.Sequence
		if r.userChanged != nil {
			close(r.userChanged)
			r.userChanged = nil
		}
	}
	if live := r.active[value.From]; live != nil && (value.FromStream || value.Discussion) && strings.TrimSpace(value.Text) != "" {
		live.spoke = true
	}
	for _, live := range notify[:n] {
		live.attention = value.Sequence
		wake(live)
	}
	r.mu.Unlock()
	event := runrecord.Event{Kind: value.Kind(), Agent: value.From, Final: value.Final, Text: value.Text, Team: value.Note}
	r.queuePublic(event)
	return value, nil
}
func wake(live *participant) {
	select {
	case live.wake <- struct{}{}:
	default:
	}
}
