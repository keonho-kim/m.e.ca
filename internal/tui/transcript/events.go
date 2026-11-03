// Purpose: Project archived runtime events and suppress duplicate task completion records.
// Pattern: Presentation projection.
// Usage: Session history passes runtime events to Formatter.Event.
// Related: internal/agent/record/events.go, internal/tui/transcript/tools.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) Event(event runrecord.Event) []viewport.Block {
	if event.Kind == "team_message" || event.Kind == "team_progress" || event.Kind == "team_notice" {
		return f.teamEvent(event)
	}
	if event.Agent != "" {
		agent := event.Agent
		event.Agent = ""
		blocks := f.Event(event)
		if len(blocks) == 0 {
			return nil
		}
		return append([]viewport.Block{{Text: "**" + agent + "**", Markdown: true}}, blocks...)
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
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied", "command_finished":
		if event.Call != nil && event.Reference != nil {
			summary := f.toolSummary(event)
			return []viewport.Block{{Text: summary, Summary: summary, DetailRef: event.Reference}}
		}
		if block := f.toolBlock(event); block.Text != "" {
			block.Summary, block.Details = f.toolSummary(event), block.Text
			return []viewport.Block{block}
		}
		return nil
	}
	if cp := event.Checkpoint; cp != nil && cp.Phase == "finished" && cp.Status == runrecord.Answered {
		return nil
	}
	if event.Kind == "thinking" {
		return []viewport.Block{Thought(f.Redact(event.Thinking))}
	}
	if event.Text != "" {
		return []viewport.Block{{Text: f.Redact(event.Text)}}
	}
	return nil
}
