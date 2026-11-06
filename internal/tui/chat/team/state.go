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
	message             messageDisplay
	current             modelapi.TokenUsage
	measured            runrecord.Throughput
	compressing, active bool
}
type State struct {
	peers    map[string]*peerDisplay
	tools    map[string]toolDisplay
	measured runrecord.Throughput
	members  []string
	names    string
}

func (s *State) Begin() {
	*s = State{peers: make(map[string]*peerDisplay), tools: make(map[string]toolDisplay)}
}
func (s *State) Finish(output *viewport.View) {
	for _, peer := range s.peers {
		output.FinishStream(peer.message.body)
	}
	*s = State{}
}
func (s *State) Count() int { return len(s.members) }

// Names returns cached active names separated by spaces in join order.
func (s *State) Names() string { return s.names }
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
	previous := output.Next
	output.AppendBlock(block)
	if output.Next == previous {
		return 0
	}
	return output.Next
}
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
	measured := s.measured
	for _, peer := range s.peers {
		current = current.Add(peer.current)
		measured = measured.Add(peer.measured)
	}
	return current, measured.Rate()
}
