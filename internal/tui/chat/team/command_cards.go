// Purpose: Refresh owned command activity without rebuilding unchanged tool output.
// Pattern: Event-loop-owned state projection.
// Usage: The session clock supplies command snapshots; terminal results release their card identities.
// Related: internal/tui/chat/team/tools.go, internal/tui/chat/team/tool_sources.go, internal/tui/transcript/commands.go.
package team

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (s *State) Commands(agent string, commands []toolapi.CommandRun, f transcript.Formatter, output *viewport.View) {
	for _, command := range commands {
		id := commandKey(agent, command.ID)
		block, found := s.tools[id]
		if !found {
			identity := command.CallID
			if command.ExecutionID != "" {
				identity = command.ExecutionID
			}
			pending := toolCallKey(agent, identity)
			block, found = s.tools[pending]
			if found {
				delete(s.tools, pending)
			}
		}
		if block.body == 0 || block.status != "" && !toolapi.CommandActive(block.status) {
			continue
		}
		if !block.paired(output) {
			block.maskVersion = commandMaskVersion(f)
			block.show(agent, false, f.CommandBlock(command), f, output)
		}
		block.refreshMask(f, output)
		block.refreshStatus(command, f, output)
		output.Update(block.header, f.TeamCommandHeader(agent, command))
		s.tools[id] = block
	}
}
