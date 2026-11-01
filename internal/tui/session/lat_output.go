// Purpose: Keep scoped LAT streams and native tool output inside their own workflow group.
// Pattern: Event-loop-owned presentation transitions.
// Usage: receiveRuntime delegates LAT events; chatLayout groups the active live suffix.
// Related: internal/tui/transcript/lat_conversation.go, internal/agent/run/lat_events.go.
package session

import (
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type latOutput struct {
	group   viewport.Group
	running bool
}

func (a *app) latGroup() viewport.Group {
	return a.latOutput.group
}

func (a *app) appendLATBlock(block viewport.Block) {
	block.Group = a.latGroup()
	a.output.AppendBlock(block)
}

func (a *app) flushLATPartial() {
	if a.latOutput.group.ID == "" {
		return
	}
	a.appendLATBlock(transcript.Thought(a.thinking))
	a.appendLATBlock(transcript.Answer(a.stream))
	a.stream, a.thinking = "", ""
}

func (a *app) receiveLAT(event *runrecord.Event) bool {
	if event.LAT == nil {
		return false
	}
	switch event.Kind {
	case "generation_started", "generation_finished", "compaction_started", "compaction_finished":
		return false // These measurements and lifecycle indicators have one owner.
	case "usage", "usage_progress":
		a.receiveTokens(*event)
		return true
	case "lat_phase_started":
		a.latOutput.group = transcript.LATGroup(*event.LAT)
		a.startLATIndicator()
		a.stream, a.thinking = "", ""
		a.clearToolActivity()
		return true
	case "lat_phase_finished":
		if event.Text != "" {
			a.flushLATPartial()
		}
		a.clearToolActivity()
		a.stopLATIndicator()
		a.stopCompression()
		a.status = ""
		if event.LAT.Phase != "review" && event.Text == "" {
			a.latOutput = latOutput{}
		}
		return true
	case "stream":
		a.stream, a.thinking = event.Text, event.Thinking
		// Hide the machine verdict while it is being generated. The localized
		// final outcome is displayed in its place after the phase has settled.
		if event.LAT.Phase == "review" && strings.HasPrefix("NO_FINDINGS", strings.TrimSpace(a.stream)) {
			a.stream = ""
		}
		return true
	case "thinking":
		a.thinking = ""
	case "progress", "response":
		if event.LAT.Phase == "review" && event.Kind == "response" {
			return true // Retain the live draft until its typed outcome replaces it.
		}
		a.stream, a.thinking = "", ""
	case "tool_started":
		a.startToolActivity(event)
		a.status = ""
		return true
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied", "command_finished":
		a.finishToolActivity(event)
		a.status = ""
	case "retry":
		a.status = event.Text
		return true
	}
	for _, block := range (transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).Event(*event) {
		a.output.AppendBlock(block)
	}
	return true
}

func (a *app) receiveLATOutcome(event *runrecord.Event) {
	f := transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}
	for _, block := range f.Event(*event) {
		if event.Kind != "lat_memory_unavailable" && a.latOutput.group.ID != "" {
			block.Group = a.latGroup()
		}
		a.output.AppendBlock(block)
	}
	if event.Kind != "lat_memory_unavailable" {
		if a.latOutput.group.ID != "" {
			a.stream, a.thinking = "", ""
		}
		a.stopLATIndicator()
		a.latOutput = latOutput{}
		a.status = ""
	}
}
