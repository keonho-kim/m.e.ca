// Purpose: Keep reasoning private and route completed progress, usage and tool observations outward.
// Pattern: Peer event adapter with explicit persistence decisions.
// Usage: Private runner callbacks forward through Peer.Emit.
// Related: internal/agent/team/publication.go, internal/agent/team/failure.go.
package team

import (
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func (p *Peer) Emit(event runrecord.Event) error {
	r := p.room
	if event.Kind == "generation_started" {
		p.presentation = event.Presentation
	}
	if event.Kind == "response" || event.Kind == "thinking" {
		return nil
	}
	if event.Kind == "stream" {
		if event.Text == "" && event.Thinking != "" {
			return nil
		}
		event.Thinking = ""
	}
	if event.Kind == "progress" && strings.TrimSpace(event.Text) != "" {
		r.mu.Lock()
		basedOn := p.DecisionCursor
		r.mu.Unlock()
		_, err := r.publish(channel.Message{Presentation: event.Presentation, From: p.ID, Text: event.Text, FromStream: true, Note: &runrecord.TeamNote{BasedOn: basedOn}})
		return err
	}
	if event.Statistics != nil {
		return nil
	}
	event.Agent = p.ID
	if event.Kind == "usage" {
		r.mu.Lock()
		r.statistics.Add(event)
		r.mu.Unlock()
	}
	observed, err := p.observe(event)
	if err != nil {
		return err
	}
	return r.deliver(event, !observed && event.Reference != nil || event.Kind == "usage")
}
