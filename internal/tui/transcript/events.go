// Purpose: Project archived runtime events and suppress duplicate task completion records.
// Pattern: Presentation projection.
// Usage: Session history passes runtime events to Formatter.Event.
// Related: internal/agent/run/events.go, internal/tui/transcript/tools.go.
package transcript

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

func (f Formatter) Event(event run.Event) []viewport.Block {
	if event.Internal {
		return nil
	}
	if event.Kind == "progress" {
		return []viewport.Block{Answer(f.Redact(event.Text))}
	}
	if event.Kind == "conclave" {
		if text := f.Conclave(event); text != "" {
			return []viewport.Block{{Text: text}}
		}
		return nil
	}
	switch event.Kind {
	case "monitor", "tool_proposed", "tool_started", "approved", "self_grounding":
		return nil
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied":
		if text := f.Tool(event); text != "" {
			return []viewport.Block{{Text: text}}
		}
		return nil
	}
	if cp := event.Checkpoint; cp != nil && cp.Phase == "finished" && (cp.Status == run.Answered || cp.Status == run.Verified) {
		return nil
	}
	if event.Kind == "thinking" {
		return []viewport.Block{{Text: f.Redact(event.Thinking), Thinking: true}}
	}
	if event.Text != "" {
		return []viewport.Block{{Text: f.Redact(event.Text)}}
	}
	return nil
}
