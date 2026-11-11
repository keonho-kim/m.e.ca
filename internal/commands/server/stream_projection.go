// Purpose: Convert serialized runtime snapshots into generation-bound draft state and ordered deltas.
// Pattern: Request-owned stream state machine with I/O outside the service mutex.
// Usage: Service.emit owns draft lifecycle; replay retains the bounded pure projections.
// Related: internal/commands/server/projection.go, internal/commands/server/draft_original.go.
package server

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	"strings"
)

type draftState struct {
	maskVersion int
	runID       string
	Draft
	textBytes, thinkingBytes int
	text, thinking           strings.Builder
}

func appendDraft(builder *strings.Builder, delta string) string {
	if remaining := maxDraftBytes - builder.Len(); remaining > 0 {
		builder.WriteString(prefix(delta, remaining))
	}
	return builder.String()
}

func (s *Service) emit(r *request, value record.Event) {
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	maskVersion := 0
	if r.conversation != nil && r.conversation.Runtime.Vault != nil {
		maskVersion = r.conversation.Runtime.Vault.RedactionVersion()
	}
	if (value.Kind == "stream" || value.Kind == "generation_started") && r.conversation != nil && r.conversation.Runtime.Vault != nil {
		s.mu.Lock()
		previous := r.drafts[value.Agent]
		changed := previous != nil && previous.maskVersion != maskVersion
		s.mu.Unlock()
		if changed {
			value.Text = r.conversation.Runtime.Vault.NewStreamMask()(value.Text, false)
			value.Thinking = r.conversation.Runtime.Vault.NewStreamMask()(value.Thinking, false)
		}
	}
	var generation uint64
	if value.Kind == "stream" || value.Kind == "generation_started" {
		s.mu.Lock()
		previous := r.drafts[value.Agent]
		reset := previous == nil || value.Kind == "generation_started" || len(value.Text) < previous.textBytes || len(value.Thinking) < previous.thinkingBytes || previous.maskVersion != maskVersion || previous.runID != value.RunID
		if reset {
			r.draftGeneration++
			generation = r.draftGeneration
		} else {
			generation = previous.Generation
		}
		s.mu.Unlock()
	}
	r.retainDraft(value, generation)
	s.mu.Lock()
	defer s.mu.Unlock()
	r.observe(value)
	if value.Usage != nil {
		stats := r.statistics()
		value.Statistics = &stats
	}
	if value.Kind != "stream" && value.Kind != "generation_started" {
		s.publish(Event{Task: r.task.ID, Record: value})
		if value.Kind == "team_notice" && value.Text == "left" {
			delete(r.drafts, value.Agent)
		}
		return
	}
	draft := r.drafts[value.Agent]
	if draft == nil {
		// Team runtime limits active participants to eight. Retired sources are
		// discarded above; this separate cap also bounds unexpected producers.
		if len(r.drafts) >= 16 {
			return
		}
		draft = &draftState{Draft: Draft{Agent: value.Agent}}
		r.drafts[value.Agent] = draft
	}
	reset := draft.Generation != generation
	if reset {
		draft.Generation = generation
		draft.textBytes, draft.thinkingBytes = 0, 0
		draft.Text, draft.Thinking, draft.Truncated = "", "", false
		draft.Committed = false
		draft.text.Reset()
		draft.thinking.Reset()
		s.publish(Event{Task: r.task.ID, Generation: draft.Generation, Reset: true, Record: record.Event{Kind: "draft_reset", Agent: value.Agent, RunID: value.RunID}})
	}
	draft.maskVersion, draft.runID = maskVersion, value.RunID
	if value.Kind == "generation_started" {
		return
	}
	// The core guarantees append-only snapshots between generation_started
	// events. Credential revision changes also reset the draft before applying offsets.
	text, thinking := value.Text[draft.textBytes:], value.Thinking[draft.thinkingBytes:]
	draft.textBytes, draft.thinkingBytes = len(value.Text), len(value.Thinking)
	draft.Text, draft.Thinking = appendDraft(&draft.text, text), appendDraft(&draft.thinking, thinking)
	draft.Truncated = len(value.Text) > maxDraftBytes || len(value.Thinking) > maxDraftBytes
	for text != "" || thinking != "" {
		// publish makes the one detached copy retained by the replay log.
		t, h := prefix(text, maxTextBytes), prefix(thinking, maxTextBytes)
		text, thinking = text[len(t):], thinking[len(h):]
		s.publish(Event{Task: r.task.ID, Generation: draft.Generation, Record: record.Event{Kind: "assistant_delta", Agent: value.Agent, RunID: value.RunID, Text: t, Thinking: h}})
	}
}
