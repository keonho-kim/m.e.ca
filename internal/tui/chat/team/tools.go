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

func (s *State) startTool(event runrecord.Event, f transcript.Formatter, output *viewport.View) {
	if event.Call == nil || catalog.IsTool(event.Call.Name) {
		return
	}
	id := key(event.Agent, event.Call.ID)
	if s.tools == nil {
		s.tools = make(map[string]int)
	}
	block := s.tools[id]
	if block == 0 {
		appendBlock(output, f.TeamHeader(event.Agent, false))
	}
	text := f.Tool(event)
	if event.Kind == "tool_waiting" {
		text += "\n" + f.Language.Text("실행 대기")
	}
	s.tools[id] = replaceBlock(output, block, viewport.Block{Text: text})
}
func (s *State) finishTool(event runrecord.Event, f transcript.Formatter, output *viewport.View) {
	if event.Call == nil || catalog.IsTool(event.Call.Name) {
		return
	}
	id := key(event.Agent, event.Call.ID)
	block := s.tools[id]
	if block == 0 {
		appendBlock(output, f.TeamHeader(event.Agent, false))
	}
	event.Agent = ""
	for _, value := range f.Event(event) {
		block = replaceBlock(output, block, value)
	}
	if event.Operation != nil && event.Operation.Process != nil && event.Operation.Process.Status == "running" {
		if s.tools == nil {
			s.tools = make(map[string]int)
		}
		s.tools[id] = block
	} else {
		delete(s.tools, id)
	}
}
func (s *State) Commands(agent string, commands []toolapi.CommandRun, f transcript.Formatter, output *viewport.View) {
	for _, command := range commands {
		block := s.tools[key(agent, command.CallID)]
		if block != 0 && command.Status == "running" {
			output.Update(block, viewport.Block{Text: f.CommandActivity([]toolapi.CommandRun{command})})
		}
	}
}
