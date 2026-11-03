// Purpose: Own only the IRC display projection and its active peer/tool identities.
// Pattern: Event-loop-owned feature state.
// Usage: The session forwards attributed events and retains mode settings separately.
// Related: internal/tui/chat/team/events.go, internal/tui/session/team_state.go.
package team

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type peerDisplay struct {
	header, stream      int
	current             modelapi.TokenUsage
	rate                *float64
	compressing, active bool
}
type State struct {
	peers   map[string]*peerDisplay
	tools   map[string]int
	totals  runrecord.UsageTotals
	members int
}

func (s *State) Begin() {
	*s = State{peers: make(map[string]*peerDisplay), tools: make(map[string]int)}
}
func (s *State) Finish(output *viewport.View) {
	for _, peer := range s.peers {
		output.FinishStream(peer.stream)
	}
	*s = State{}
}
func (s *State) Count() int { return s.members }
func (s *State) peer(id string) *peerDisplay {
	if s.peers == nil {
		s.peers = make(map[string]*peerDisplay)
	}
	peer := s.peers[id]
	if peer == nil {
		peer = &peerDisplay{}
		s.peers[id] = peer
	}
	return peer
}
func appendBlock(output *viewport.View, block viewport.Block) int {
	output.AppendBlock(block)
	return output.Next
}
func replaceBlock(output *viewport.View, id int, block viewport.Block) int {
	if id != 0 && output.Update(id, block) {
		return id
	}
	return appendBlock(output, block)
}
func key(agent, call string) string { return agent + "/" + call }
func (s *State) Compressing() bool {
	for _, peer := range s.peers {
		if peer.compressing {
			return true
		}
	}
	return false
}
func (s *State) CurrentUsage() (modelapi.TokenUsage, *float64) {
	var current modelapi.TokenUsage
	measured := float64(s.totals.MeasuredMillis)
	for _, peer := range s.peers {
		current = current.Add(peer.current)
		if peer.rate != nil && *peer.rate > 0 {
			measured += float64(peer.current.Output) / (*peer.rate) * 1000
		}
	}
	if measured <= 0 {
		return current, nil
	}
	rate := float64(s.totals.Usage.Output+current.Output) * 1000 / measured
	return current, &rate
}
