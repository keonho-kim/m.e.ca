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
	if e.Agent != "" {
		a.receiveTeam(e)
		return
	}
	switch e.Kind {
	case "image_input_unavailable":
		a.connection.rejectImages()
		a.status = a.language.Text("Image input is unavailable for this connection")
		return
	case "user_message":
		if len(a.compose.steeringDrafts) > 0 {
			a.compose.steeringDrafts[0] = ""
			a.compose.steeringDrafts = a.compose.steeringDrafts[1:]
		}
		a.chat.output.Append((transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).User(e.Text), false)
		a.compose.rememberInput(e.Text)
		return

	case "generation_started":
		a.chat.tokenCurrent = modelapi.TokenUsage{}
		a.chat.currentMeasurement = runrecord.Throughput{}
		return
	case "generation_finished":
		return
	case "compaction_started":
		a.startCompression()
		return
	case "compaction_finished", "compaction":
		a.stopCompression()
		return
	}
	if e.Kind == "tool_proposed" {
		return
	}
	if e.Statistics != nil {
		a.chat.receiveTranscript(e, transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact})
		return
	}
	if e.Kind == "retry" {
		a.status = e.Text
		return
	}
	if e.Kind == "usage_progress" || e.Kind == "usage" {
		a.chat.receiveTokens(*e)
		return
	}
	switch e.Kind {
	case "tool_started":
		a.startToolActivity(e)
		a.status = ""
		return
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied", "command_finished":
		a.finishToolActivity(e)
		a.status = ""
		for _, block := range (transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).Event(*e) {
			a.chat.output.AppendBlock(block)
		}
		return
	case "approved":
		return
	}
	a.chat.receiveTranscript(e, transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact})
}
