// Purpose: Register atomic public contributions and expose current work to private model decisions.
// Pattern: Room-owned state with serialized public publication.
// Usage: Team-only tools and executor admission call Peer work methods.
// Related: internal/agent/team/contribution/state.go, internal/commands/teamrequest/peer.go.
package team

import (
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (p *Peer) SetWork(summary string, paths []string) (contribution.Scope, error) {
	r := p.room
	r.logMu.Lock()
	defer r.logMu.Unlock()
	if err := r.ctx.Err(); err != nil {
		return contribution.Scope{}, err
	}
	r.mu.Lock()
	live := r.active[p.ID]
	if live == nil || live.closed || live.leaving {
		r.mu.Unlock()
		return contribution.Scope{}, fmt.Errorf("this team member is no longer active")
	}
	if r.work.Matches(p.ID, summary, paths) {
		value := r.work.Scope(p.ID)
		if len(paths) == 0 {
			value = contribution.Scope{Member: p.ID, Summary: summary, Paths: []string{}}
		}
		r.mu.Unlock()
		return value, nil
	}
	err := r.work.Set(p.ID, summary, paths)
	if err != nil {
		r.mu.Unlock()
		return contribution.Scope{}, &toolapi.Failure{Code: "team_work_conflict", Detail: err.Error(), Next: "Read the named teammate's work. Agree on a handoff publicly or take a different nonoverlapping part; do not retry the same claim unchanged."}
	}
	r.workView = workProjection{}
	value := r.work.Scope(p.ID)
	if len(paths) == 0 {
		value = contribution.Scope{Member: p.ID, Summary: summary, Paths: []string{}}
	}
	r.mu.Unlock()
	var text strings.Builder
	size := len(summary)
	for _, path := range paths {
		size += len(path) + 3
	}
	text.Grow(size)
	text.WriteString(summary)
	for _, path := range paths {
		text.WriteString("\n  ")
		text.WriteString(path)
	}
	if _, err := r.appendPublic(channel.Message{From: p.ID, Text: text.String(), Declaration: true}); err != nil {
		return contribution.Scope{}, err
	}
	r.mu.Lock()
	live.spoke = true
	r.mu.Unlock()
	return value, nil
}

// BeginDecision captures the last conversation actually admitted to this model call.
func (p *Peer) BeginDecision() {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	p.DecisionCursor = p.Cursor
	if live := r.active[p.ID]; live != nil {
		live.reviewed = p.Cursor
	}
	through := p.Cursor
	for _, live := range r.active {
		if !live.closed && !live.leaving {
			through = min(through, live.reviewed)
		}
	}
	r.work.Prune(through)
}

func (p *Peer) CheckWork(paths []string) error {
	r := p.room
	r.mu.Lock()
	err := r.work.Check(p.ID, paths, p.DecisionCursor)
	r.mu.Unlock()
	if err == nil {
		return nil
	}
	return &toolapi.Failure{Code: "team_work_conflict", Detail: "Nothing executed. " + err.Error(), Next: "Read the current team work and actual results, coordinate by name, and regenerate only the still-needed operation."}
}
