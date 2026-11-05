// Purpose: Maintain team tool cards from prepared events and owned command snapshots.
// Pattern: Source-keyed presentation projection.
// Usage: State.Apply and the session clock update cards through the same identity map.
// Related: internal/tui/chat/team/events.go, internal/tui/transcript/tools.go.
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
	command  transcript.CommandView
	terminal bool
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
	text := f.Tool(event)
	if event.Kind == "tool_waiting" {
		text += "\n" + f.Language.Text("실행 대기")
	}
	block.show(event.Agent, false, viewport.Block{Text: text}, f, output)
	s.tools[id] = block
}
func (s *State) finishTool(event runrecord.Event, f transcript.Formatter, output *viewport.View) {
	if event.Call == nil || catalog.IsTool(event.Call.Name) {
		return
	}
	id := toolCallKey(event.Agent, event.Call.ID)
	command := event.Call.Name == "execute_command" && event.Operation != nil && event.Operation.Process != nil
	if command {
		owned := commandKey(event.Agent, event.Operation.Process.CommandID)
		if _, exists := s.tools[owned]; exists {
			id = owned
		}
	}
	block := s.tools[id]
	if command && toolapi.CommandActive(event.Operation.Process.Status) && block.terminal {
		return
	}
	agent := event.Agent
	event.Agent = ""
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
		delete(s.tools, id)
	}
}
func (s *State) Commands(agent string, commands []toolapi.CommandRun, f transcript.Formatter, output *viewport.View) {
	for _, command := range commands {
		id := commandKey(agent, command.ID)
		block, found := s.tools[id]
		if !found {
			pending := toolCallKey(agent, command.CallID)
			block, found = s.tools[pending]
			if found {
				delete(s.tools, pending)
			}
		}
		if block.body != 0 && !block.terminal {
			block.show(agent, false, viewport.Block{Text: block.command.Render(command, f)}, f, output)
			block.terminal = !toolapi.CommandActive(command.Status)
			s.tools[id] = block
		}
	}
}

func toolCallKey(agent, call string) string { return agent + "/call:" + call }
func commandKey(agent, id string) string    { return agent + "/command:" + id }
