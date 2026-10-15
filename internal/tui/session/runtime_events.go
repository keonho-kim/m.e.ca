// Purpose: Apply runtime notifications to active transcript, token and streaming display state.
// Pattern: Presentation state transition.
// Usage: The session event dispatcher invokes receiveRuntime on its event-loop thread.
// Related: internal/tui/session/events.go, internal/tui/transcript/events.go.
package session

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
)

func (a *app) receiveRuntime(e *run.Event) {
	if e.Internal || e.Kind == "monitor" || e.Kind == "tool_proposed" {
		return
	}
	if e.Kind == "conclave" {
		formatter := transcript.Formatter{Redact: a.opts.Vault.Redact}
		if e.Conclave != nil && e.Conclave.Passed && e.Call != nil {
			a.status = e.Call.Name + " · Conclave Pass"
		} else {
			a.status = ""
			a.output.Append(formatter.Conclave(*e), false)
		}
		return
	}
	if e.Kind == "retry" {
		a.status = e.Text
		return
	}
	if e.Kind == "usage_progress" || e.Kind == "usage" {
		if e.Usage != nil {
			a.tokenCurrent = *e.Usage
			if e.Kind == "usage" {
				a.tokenTotal = a.displayTokens()
				a.tokenCurrent = modelapi.TokenUsage{}
			}
		}
		return
	}
	switch e.Kind {
	case "tool_started":
		a.status = (transcript.Formatter{Redact: a.opts.Vault.Redact}).Tool(*e)
		return
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied":
		a.status = ""
		a.output.Append((transcript.Formatter{Redact: a.opts.Vault.Redact}).Tool(*e), false)
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
		a.output.Append(e.Thinking, true)
		return
	}
	if e.Kind == "stream" {
		a.stream, a.thinking = e.Text, e.Thinking
	} else if e.Kind == "response" {
		a.stream, a.thinking = "", ""
		a.output.Append(e.Thinking, true)
		_, proposalErr := planning.ParsePlanProposal(e.Text)
		if !a.planMode || proposalErr != nil {
			a.output.AppendBlock(transcript.Answer(e.Text))
		}
	} else {
		a.output.Append(e.Text, false)
	}
}
