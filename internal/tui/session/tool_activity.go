// Purpose: Preserve each pending tool independently from notices and observed command lifetimes.
// Pattern: Event-loop-owned presentation state.
// Usage: Runtime notifications update pending rows; chatLayout projects them with command snapshots.
// Related: internal/tui/session/runtime_events.go, internal/tui/session/command_activity.go.
package session

import (
	"slices"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

type pendingTool struct{ callID, text string }

type toolActivity struct {
	pending  []pendingTool
	commands []toolapi.CommandRun
	text     string
	dirty    bool
}

func toolCallID(event *runrecord.Event) string {
	if event.CallID != "" || event.Call == nil {
		return event.CallID
	}
	return event.Call.ID
}

func (a *app) startToolActivity(event *runrecord.Event) {
	if event.Call == nil || a.worker.stopping() {
		return
	}
	entry := pendingTool{callID: toolCallID(event), text: (transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).Tool(*event)}
	if index := slices.IndexFunc(a.toolActivity.pending, func(p pendingTool) bool { return p.callID == entry.callID }); index >= 0 {
		a.toolActivity.pending[index] = entry
	} else {
		a.toolActivity.pending = append(a.toolActivity.pending, entry)
	}
	a.toolActivity.dirty = true
}

func (a *app) finishToolActivity(event *runrecord.Event) {
	id := toolCallID(event)
	if index := slices.IndexFunc(a.toolActivity.pending, func(p pendingTool) bool { return p.callID == id }); index >= 0 {
		a.toolActivity.pending = slices.Delete(a.toolActivity.pending, index, index+1)
		a.toolActivity.dirty = true
	}
	a.updateCommandActivity()
}

func (a *app) liveToolActivity() string {
	if !a.toolActivity.dirty {
		return a.toolActivity.text
	}
	f := transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}
	var text strings.Builder
	for _, pending := range a.toolActivity.pending {
		// A running command snapshot replaces its start row without duplicating it.
		index := slices.IndexFunc(a.toolActivity.commands, func(run toolapi.CommandRun) bool {
			return pending.callID != "" && run.CallID == pending.callID && run.Status == "running"
		})
		if index >= 0 {
			text.WriteString(f.CommandActivity(a.toolActivity.commands[index : index+1]))
		} else {
			text.WriteString(pending.text)
		}
	}
	for index, run := range a.toolActivity.commands {
		if !slices.ContainsFunc(a.toolActivity.pending, func(p pendingTool) bool { return run.CallID != "" && p.callID == run.CallID }) {
			text.WriteString(f.CommandActivity(a.toolActivity.commands[index : index+1]))
		}
	}
	a.toolActivity.text, a.toolActivity.dirty = text.String(), false
	return a.toolActivity.text
}

func (a *app) clearToolActivity() { a.toolActivity = toolActivity{} }
