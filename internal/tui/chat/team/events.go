// Purpose: Project public peer events into stable transcript blocks without performing session I/O.
// Pattern: Deterministic event reducer.
// Usage: Session receiveTeam forwards display events and handles application side effects.
// Related: internal/tui/chat/team/state.go, internal/tui/chat/team/tools.go.
package team

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (s *State) Apply(event runrecord.Event, f transcript.Formatter, output *viewport.View) {
	switch event.Kind {
	case "team_notice":
		peer := s.peer(event.Agent)
		joined := event.Text == "joined"
		if joined && !peer.active {
			s.members++
		}
		if !joined && peer.active {
			s.members--
		}
		peer.active = joined
		for _, block := range f.Event(event) {
			appendBlock(output, block)
		}
		if !joined {
			output.FinishStream(peer.message.body)
			delete(s.peers, event.Agent)
		}
	case "generation_started":
		peer := s.peer(event.Agent)
		output.FinishStream(peer.message.body)
		peer.message = messageDisplay{}
		peer.measured = runrecord.Throughput{}
		peer.current = modelapi.TokenUsage{}
	case "stream":
		peer := s.peer(event.Agent)
		peer.message.show(event.Agent, false, viewport.Block{Text: event.Text, Markdown: true, Streaming: true}, f, output)
	case "team_message", "team_progress":
		blocks := f.Event(event)
		if len(blocks) != 2 {
			return
		}
		if event.Kind == "team_progress" {
			peer := s.peer(event.Agent)
			peer.message.show(event.Agent, event.Final, blocks[1], f, output)
			peer.message = messageDisplay{}
		} else {
			appendBlock(output, blocks[0])
			appendBlock(output, blocks[1])
		}
	case "compaction_started":
		s.peer(event.Agent).compressing = true
	case "compaction_finished", "compaction":
		s.peer(event.Agent).compressing = false
	case "usage_progress":
		if event.Usage != nil {
			peer := s.peer(event.Agent)
			peer.current = event.Usage.Clone()
			peer.measured = runrecord.MeasuredThroughput(event)
		}
	case "usage":
		s.measured = s.measured.Add(runrecord.MeasuredThroughput(event))
		peer := s.peer(event.Agent)
		peer.current = modelapi.TokenUsage{}
		peer.measured = runrecord.Throughput{}
	case "tool_started", "tool_waiting":
		s.startTool(event, f, output)
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied", "command_finished":
		s.finishTool(event, f, output)
	}
}
