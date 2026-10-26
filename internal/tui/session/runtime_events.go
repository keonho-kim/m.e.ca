// Purpose: Apply runtime notifications to active transcript, token and streaming display state.
// Pattern: Presentation state transition.
// Usage: The session event dispatcher invokes receiveRuntime on its event-loop thread.
// Related: internal/tui/session/events.go, internal/tui/transcript/events.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func (a *app) receiveRuntime(e *runrecord.Event) {
	switch e.Kind {
	case "user_message":
		if len(a.steeringDrafts) > 0 {
			a.steeringDrafts[0] = ""
			a.steeringDrafts = a.steeringDrafts[1:]
		}
		a.output.Append((transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).User(e.Text), false)
		a.rememberInput(e.Text)
		return

	case "generation_started":
		a.tokenCurrent = modelapi.TokenUsage{}
		a.tokensPerSecond = nil
		return
	case "compaction_started":
		a.startCompression()
		return
	case "compaction_finished", "compaction":
		a.stopCompression()
		return
	}
	if e.Internal || e.Kind == "tool_proposed" {
		return
	}
	if e.Statistics != nil {
		blocks := (transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).Event(*e)
		a.runSummary = &blocks[0]
		return
	}
	if e.Kind == "retry" {
		a.status = e.Text
		return
	}
	if e.Kind == "usage_progress" || e.Kind == "usage" {
		a.receiveTokens(*e)
		return
	}
	switch e.Kind {
	case "tool_started":
		a.status = (transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).Tool(*e)
		return
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied":
		a.status = ""
		for _, block := range (transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).Event(*e) {
			a.output.AppendBlock(block)
		}
		return
	case "approved":
		return
	}
	if e.Kind == "progress" {
		a.stream, a.thinking = "", ""
		a.output.AppendBlock(transcript.Answer(e.Text))
		return
	}
	if e.Kind == "thinking" {
		a.thinking = ""
		a.output.AppendBlock(transcript.Thought(e.Thinking))
		return
	}
	if e.Kind == "stream" {
		a.stream, a.thinking = e.Text, e.Thinking
	} else if e.Kind == "response" {
		a.stream, a.thinking = "", ""
		a.output.AppendBlock(transcript.Thought(e.Thinking))
		a.output.AppendBlock(transcript.Answer(e.Text))
	} else {
		a.output.Append(e.Text, false)
	}
}
