// Purpose: Connect team tool events and command controls to their owned display cards.
// Pattern: Source-keyed presentation projection.
// Usage: State.Apply and the session clock update cards through the same identity map.
// Related: internal/tui/chat/team/command_cards.go, internal/tui/chat/team/events.go, internal/tui/transcript/tools.go.
package team

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/team/catalog"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type toolDisplay struct {
	messageDisplay
	status      string
	maskVersion int
}

func (s *State) startTool(event runrecord.Event, f transcript.Formatter, output *viewport.View) {
	if event.Call == nil || catalog.IsTool(event.Call.Name) {
		return
	}
	id := toolCallKey(event.Agent, event.Call.ID)
	if s.tools == nil {
		s.tools = make(map[string]toolDisplay)
	}
	block := s.tools[id]
	block.maskVersion = commandMaskVersion(f)
	block.show(event.Agent, false, f.PendingTool(event), f, output)
	s.tools[id] = block
}
func (s *State) finishTool(event runrecord.Event, f transcript.Formatter, output *viewport.View) {
	if event.Call == nil || catalog.IsTool(event.Call.Name) {
		return
	}
	id := toolCallKey(event.Agent, event.Call.ID)
	command := event.Operation != nil && event.Operation.Process != nil && event.Operation.Process.CommandID != ""
	if command {
		owned := commandKey(event.Agent, event.Operation.Process.CommandID)
		if _, exists := s.tools[owned]; exists {
			if id != owned {
				pending := s.tools[id]
				pending.discard(output)
				delete(s.tools, id)
			}
			id = owned
		}
	}
	block := s.tools[id]
	if command && toolapi.CommandActive(event.Operation.Process.Status) && block.status != "" && !toolapi.CommandActive(block.status) {
		return
	}
	agent := event.Agent
	event.Agent = ""
	block.maskVersion = commandMaskVersion(f)
	if command {
		block.status = event.Operation.Process.Status
	}
	for _, value := range f.Event(event) {
		block.show(agent, false, value, f, output)
	}
	if command && toolapi.CommandActive(event.Operation.Process.Status) {
		if s.tools == nil {
			s.tools = make(map[string]toolDisplay)
		}
		delete(s.tools, id)
		s.tools[commandKey(agent, event.Operation.Process.CommandID)] = block
	} else {
		output.Update(block.header, f.TeamHeader(agent, false))
		delete(s.tools, id)
	}
}

func toolCallKey(agent, call string) string { return agent + "/call:" + call }
func commandKey(agent, id string) string    { return agent + "/command:" + id }
