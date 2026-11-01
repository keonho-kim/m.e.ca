// Purpose: Project scoped LAT messages and native tool outcomes into the same grouped conversation.
// Pattern: Presentation adapter over canonical side-session records.
// Usage: Live and restored LAT records share LATGroup and latConversation.
// Related: internal/tui/session/lat_output.go, internal/tui/transcript/events.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

// LATGroup groups one workflow phase without retaining its conversation source.
func LATGroup(scope runrecord.LATRun) viewport.Group {
	return viewport.Group{ID: scope.ParentRunID + ":" + scope.Phase, Title: "Look At This"}
}

func (f Formatter) latConversation(event runrecord.Event) []viewport.Block {
	scope := *event.LAT
	var blocks []viewport.Block
	if event.Kind == "lat_message" {
		message := event.Message
		if message == nil || message.Role != "assistant" {
			return nil
		}
		// The typed parent outcome owns the final review feedback and replaces
		// the machine-only NO_FINDINGS marker. Intermediate commentary remains.
		if scope.Phase == "review" && len(message.Calls) == 0 && !message.Incomplete {
			return nil
		}
		if message.Reasoning != "" {
			blocks = append(blocks, Thought(f.Redact(message.Reasoning)))
		}
		if message.Content != "" {
			blocks = append(blocks, Answer(f.Redact(message.Content)))
		}
	} else {
		switch event.Kind {
		case "thinking", "progress", "response", "tool_succeeded", "tool_failed", "tool_interrupted", "denied", "command_finished":
			event.Internal, event.LAT = false, nil
			if event.Kind == "response" {
				blocks = []viewport.Block{Answer(f.Redact(event.Text))}
			} else {
				blocks = f.Event(event)
			}
		default:
			return nil
		}
	}
	for i := range blocks {
		blocks[i].Group = LATGroup(scope)
	}
	return blocks
}
