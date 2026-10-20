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
	if event.Statistics != nil {
		return []viewport.Block{f.runSummary(*event.Statistics)}
	}
	if event.Kind == "progress" {
		return []viewport.Block{Answer(f.Redact(event.Text))}
	}
	switch event.Kind {
	case "generation_started", "compaction", "compaction_started", "compaction_finished", "tool_proposed", "tool_started", "approved":
		return nil
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied":
		if text := f.Tool(event); text != "" {
			block := viewport.Block{Text: text, Summary: f.toolSummary(event), Details: text}
			return []viewport.Block{block}
		}
		return nil
	}
	if cp := event.Checkpoint; cp != nil && cp.Phase == "finished" && cp.Status == run.Answered {
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
